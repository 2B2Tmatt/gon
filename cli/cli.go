package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"gon/lock"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Operation string

// cli enums
const (
	OpInit    Operation = "init"
	OpAdd     Operation = "add"
	OpInstall Operation = "install"
)

type GonFile struct {
	Name         string            `json:"name"`
	Version      int               `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

type Cli struct {
	Op     Operation
	Args   []string
	Lf     *lock.Lockfile
	Gf     *GonFile
	Client *http.Client
}

type Command func(*Cli) error

func LoadCli() (*Cli, error) {
	cli := Cli{}
	cli.Args = os.Args[1:]
	if len(cli.Args) < 1 {
		return nil, errors.New("no command entered, use -help for options")
	}
	cli.Op = Operation(cli.Args[0])

	Operations := map[Operation]struct{}{
		OpInit:    {},
		OpAdd:     {},
		OpInstall: {},
	}
	_, supported := Operations[cli.Op]
	if !supported {
		return nil, errors.New("unsupported operation")
	}
	_, err := os.Stat("gon.json")
	if os.IsNotExist(err) {
		cli.Gf = &GonFile{
			Name:         "my-project",
			Version:      1,
			Dependencies: make(map[string]string, 0),
		}
	} else if err != nil {
		return nil, err
	} else {
		bytes, err := os.ReadFile("gon.json")
		if err != nil {
			return nil, err
		}
		cli.Gf = &GonFile{}
		err = json.Unmarshal(bytes, cli.Gf)
		if err != nil {
			return nil, err
		}
		if cli.Gf.Name == "" {
			cli.Gf.Name = "my-project"
		}
		if cli.Gf.Version <= 0 {
			cli.Gf.Version = 1
		}
		if cli.Gf.Dependencies == nil {
			cli.Gf.Dependencies = make(map[string]string, 0)
		}
	}
	cli.Client = &http.Client{
		Timeout: time.Second * time.Duration(15),
	}
	return &cli, nil
}

func (cli *Cli) Dispatch() error {
	Commands := map[Operation]Command{
		OpInit:    runInit,
		OpAdd:     runAdd,
		OpInstall: runInstall,
	}
	cmd, supported := Commands[cli.Op]
	if !supported {
		return errors.New("unsupported operation")
	}
	err := cmd(cli)
	if err != nil {
		return err
	}
	return nil
}

func runInit(cli *Cli) error {
	_, err := os.Stat("gon.json")
	if os.IsNotExist(err) {
		if len(cli.Args) >= 2 {
			cli.Gf.Name = cli.Args[1]
		}
		bytes, err := json.MarshalIndent(cli.Gf, "", " ")
		if err != nil {
			return err
		}
		err = os.WriteFile("gon.json", bytes, 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func runAdd(cli *Cli) error {
	_, err := os.Stat("gon.json")
	if os.IsNotExist(err) {
		err := cli.UpdateGon()
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	bytes, err := os.ReadFile("gon.json")
	if err != nil {
		return err
	}
	if len(cli.Args) >= 2 {
		pkg, version, verIsSpecified := strings.Cut(cli.Args[1], "@")
		var gf GonFile
		json.Unmarshal(bytes, &gf)

		var resp *http.Response
		var err error
		if verIsSpecified {
			resp, err = cli.Client.Get(fmt.Sprintf("https://registry.npmjs.org/%s/%s", pkg, version))
		} else {
			resp, err = cli.Client.Get(fmt.Sprintf("https://registry.npmjs.org/%s/latest", pkg))
		}
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return errors.New("package does not exist in registry")
		}
		var responsePkg Package
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		json.Unmarshal(bytes, &responsePkg)
		gf.Dependencies[cli.Args[1]] = responsePkg.Version
		cli.Gf = &gf
		err = cli.UpdateGon()
		if err != nil {
			return err
		}
	}
	fmt.Println("Add dispatch")
	return nil
}

func runInstall(cli *Cli) error {
	fmt.Println("Install dispatch")
	return nil
}

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    Distro `json:"dist"`
}

type Distro struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"`
}

func (cli *Cli) UpdateGon() error {
	bytes, err := json.MarshalIndent(cli.Gf, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile("gon.json", bytes, 0755)
	if err != nil {
		return err
	}
	return nil
}
