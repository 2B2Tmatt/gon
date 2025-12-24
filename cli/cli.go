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
	OpHelp    Operation = "help"
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
		OpHelp:    {},
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
		OpHelp:    runHelp,
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

func runHelp(cli *Cli) error {
	fmt.Println("--------------------------------------\n")
	fmt.Println("Commands\n")
	fmt.Println("'init [name]'- initilizes gon.json")
	fmt.Println("'add [package]` adds package(if valid) to gon.json")
	fmt.Println("'install' resolves gon.json -> gon-lock.json and fetches packages\n")
	fmt.Println("--------------------------------------")
	return nil
}

func runInit(cli *Cli) error {
	_, err := os.Stat("gon.json")
	if os.IsNotExist(err) {
		if len(cli.Args) >= 2 {
			cli.Gf.Name = cli.Args[1]
		}
		err := cli.UpdateGon()
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
		gf.Dependencies[pkg] = responsePkg.Version
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
	cli.ResolveAll()
	return nil
}

type Package struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dist         Distro            `json:"dist"`
	Dependencies map[string]string `json:"dependencies"`
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

func (cli *Cli) ResolveAll() error {
	bytes, err := os.ReadFile("gon.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, cli.Gf)
	if err != nil {
		return err
	}
	bytes, err = os.ReadFile("gon-lock.json")
	var lf lock.Lockfile
	if os.IsNotExist(err) {
		lf = lock.Lockfile{
			LockFileVersion:  1,
			Registry:         "https://registry.npmjs.org",
			RootDependencies: make(map[string]string),
			Packages:         make(map[string]*lock.Package),
		}
	} else if err != nil {
		return err
	} else {
		err = json.Unmarshal(bytes, &lf)
		if err != nil {
			return err
		}
	}
	cli.Lf = &lf
	for pkgName, ver := range cli.Gf.Dependencies {
		err = cli.Resolve(pkgName, ver, cli.Lf)
		if err != nil {
			return err
		}
	}
	fmt.Println("Resolved all Packages")
	fmt.Println("Installing")
	bytes, err = json.MarshalIndent(cli.Lf, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile("gon-lock.json", bytes, 0755)
	if err != nil {
		return err
	}

	fmt.Println("Finished writing")

	return nil
}

func (cli *Cli) Resolve(pkgName, ver string, lf *lock.Lockfile) error {
	resp, err := cli.Client.Get(fmt.Sprintf("https://registry.npmjs.org/%s/%s", pkgName, ver))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(pkgName + " was not found in the registry")
	}
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var pkgResp Package
	err = json.Unmarshal(bytes, &pkgResp)
	if err != nil {
		return err
	}
	_, isRootDep := cli.Gf.Dependencies[pkgName]
	pkgID := fmt.Sprintf("%s@%s", pkgName, ver)
	if isRootDep {
		lf.RootDependencies[pkgName] = pkgID
	}

	lockPkg, pkgInLock := cli.Lf.Packages[pkgID]

	if pkgInLock {
		lockPkg.Version = ver
		lockPkg.TarballURL = pkgResp.Dist.Tarball
		lockPkg.Integrity = pkgResp.Dist.Integrity
		if len(pkgResp.Dependencies) != 0 {
			for depPkg, verRange := range pkgResp.Dependencies {
				ver, err := ResolveVersion(cli.Client, depPkg, verRange)
				if err != nil {
					return err
				}
				lockPkg.Deps[depPkg] = fmt.Sprintf("%s@%s", depPkg, ver)
			}
		}
	} else {
		pkgDeps := make(map[string]string)
		if len(pkgResp.Dependencies) != 0 {
			for depPkg, verRange := range pkgResp.Dependencies {
				ver, err := ResolveVersion(cli.Client, depPkg, verRange)
				if err != nil {
					return err
				}
				pkgDeps[depPkg] = fmt.Sprintf("%s@%s", depPkg, ver)
			}
		}
		cli.Lf.Packages[pkgID] = &lock.Package{
			Name:       pkgName,
			Version:    ver,
			TarballURL: pkgResp.Dist.Tarball,
			Integrity:  pkgResp.Dist.Integrity,
			Deps:       pkgDeps,
		}
	}
	for pkName, pkID := range cli.Lf.Packages[pkgID].Deps {
		splitId := strings.Split(pkID, "@")
		err = cli.Resolve(pkName, splitId[1], cli.Lf)
		if err != nil {
			return err
		}
	}
	return nil
}

func ResolveVersion(client *http.Client, pkgName, ver string) (string, error) {
	if strings.Contains(ver, "^") || strings.Contains(ver, "~") {
		resp, err := client.Get(fmt.Sprintf("https://registry.npmjs.org/%s/latest", pkgName))
		if err != nil {
			return "", err
		}
		if resp.StatusCode != http.StatusOK {
			return "", errors.New("pkg: " + pkgName + " not found in registry")
		}
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		var pkg Package
		err = json.Unmarshal(bytes, &pkg)
		if err != nil {
			return "", err
		}

		return pkg.Version, nil
	}

	return ver, nil
}
