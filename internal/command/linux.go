package command

import (
	"fmt"
	"runtime"

	"github.com/fyne-io/fyne-cross/internal/log"
	"github.com/fyne-io/fyne-cross/internal/volume"
)

const (
	// linuxOS it the linux OS name
	linuxOS = "linux"
	// linuxImage is the fyne-cross image for the Linux OS
	linuxImageAmd64 = "ghcr.io/herugen/fyne-cross-images-linux:latest"
	linuxImage386   = "ghcr.io/herugen/fyne-cross-images-linux:latest"
	linuxImageArm64 = "ghcr.io/herugen/fyne-cross-images-linux:latest"
	linuxImageArm   = "ghcr.io/herugen/fyne-cross-images-linux:latest"
)

var (
	// linuxArchSupported defines the supported target architectures on linux
	linuxArchSupported = []Architecture{ArchAmd64, Arch386, ArchArm, ArchArm64}
)

// linux build and package the fyne app for the linux OS
type linux struct {
	Images         []containerImage
	defaultContext Context
}

var _ platformBuilder = (*linux)(nil)
var _ Command = (*linux)(nil)

func NewLinuxCommand() *linux {
	return &linux{}
}

func (cmd *linux) Name() string {
	return "linux"
}

// Description returns the command description
func (cmd *linux) Description() string {
	return "Build and package a fyne application for the linux OS"
}

func (cmd *linux) Run() error {
	return commonRun(cmd.defaultContext, cmd.Images, cmd)
}

// Parse parses the arguments and set the usage for the command
func (cmd *linux) Parse(args []string) error {
	commonFlags, err := newCommonFlags()
	if err != nil {
		return err
	}

	flags := &linuxFlags{
		CommonFlags: commonFlags,
		TargetArch:  &targetArchFlag{runtime.GOARCH},
	}
	flagSet.Var(flags.TargetArch, "arch", fmt.Sprintf(`List of target architecture to build separated by comma. Supported arch: %s`, linuxArchSupported))

	flagSet.Usage = cmd.Usage
	flagSet.Parse(args)

	err = cmd.setupContainerImages(flags, flagSet.Args())
	return err
}

// Run runs the command
func (cmd *linux) Build(image containerImage) (string, error) {
	err := prepareIcon(cmd.defaultContext, image)
	if err != nil {
		return "", err
	}

	//
	// package
	//
	log.Info("[i] Packaging app...")
	packageName := fmt.Sprintf("%s.tar.xz", cmd.defaultContext.Name)

	if cmd.defaultContext.Release {
		// Release mode
		err = fyneRelease(cmd.defaultContext, image)
	} else {
		// Build mode
		err = fynePackage(cmd.defaultContext, image)
	}
	if err != nil {
		return "", fmt.Errorf("could not package the Fyne app: %v", err)
	}
	image.Run(cmd.defaultContext.Volume, options{}, []string{
		"mv",
		volume.JoinPathContainer(cmd.defaultContext.WorkDirContainer(), packageName),
		volume.JoinPathContainer(cmd.defaultContext.TmpDirContainer(), image.ID(), packageName),
	})

	// Extract the resulting executable from the tarball
	image.Run(cmd.defaultContext.Volume,
		options{WorkDir: volume.JoinPathContainer(cmd.defaultContext.BinDirContainer(), image.ID())},
		[]string{"tar", "-xf",
			volume.JoinPathContainer(cmd.defaultContext.TmpDirContainer(), image.ID(), packageName),
			"--strip-components=3", "usr/local/bin"})

	return packageName, nil
}

// Usage displays the command usage
func (cmd *linux) Usage() {
	data := struct {
		Name        string
		Description string
	}{
		Name:        cmd.Name(),
		Description: cmd.Description(),
	}

	template := `
Usage: fyne-cross {{ .Name }} [options] [package]

{{ .Description }}

Options:
`

	printUsage(template, data)
	flagSet.PrintDefaults()
}

// linuxFlags defines the command-line flags for the linux command
type linuxFlags struct {
	*CommonFlags

	// TargetArch represents a list of target architecture to build on separated by comma
	TargetArch *targetArchFlag
}

// setupContainerImages returns the command ContainerImages for a linux target
func (cmd *linux) setupContainerImages(flags *linuxFlags, args []string) error {
	targetArch, err := targetArchFromFlag(*flags.TargetArch, linuxArchSupported)
	if err != nil {
		return fmt.Errorf("could not make build context for %s OS: %s", linuxOS, err)
	}

	ctx, err := makeDefaultContext(flags.CommonFlags, args)
	if err != nil {
		return err
	}

	cmd.defaultContext = ctx
	runner, err := newContainerEngine(ctx)
	if err != nil {
		return err
	}

	for _, arch := range targetArch {
		var image containerImage

		switch arch {
		case ArchAmd64:
			image = runner.createContainerImage(arch, linuxOS, overrideDockerImage(flags.CommonFlags, linuxImageAmd64))
			image.SetEnv("GOARCH", "amd64")
			image.SetEnv("CC", "zig cc -target x86_64-linux-gnu -isystem /usr/include -L/usr/lib/x86_64-linux-gnu")
			image.SetEnv("CXX", "zig c++ -target x86_64-linux-gnu -isystem /usr/include -L/usr/lib/x86_64-linux-gnu")
		case Arch386:
			image = runner.createContainerImage(arch, linuxOS, overrideDockerImage(flags.CommonFlags, linuxImage386))
			image.SetEnv("GOARCH", "386")
			image.SetEnv("CC", "zig cc -target x86-linux-gnu -isystem /usr/include -L/usr/lib/i386-linux-gnu")
			image.SetEnv("CXX", "zig c++ -target x86-linux-gnu -isystem /usr/include -L/usr/lib/i386-linux-gnu")
		case ArchArm:
			image = runner.createContainerImage(arch, linuxOS, overrideDockerImage(flags.CommonFlags, linuxImageArm))
			image.SetEnv("GOARCH", "arm")
			image.SetEnv("GOARM", "7")
			image.SetEnv("CC", "zig cc -target arm-linux-gnueabihf -isystem /usr/include -L/usr/lib/arm-linux-gnueabihf")
			image.SetEnv("CXX", "zig c++ -target arm-linux-gnueabihf -isystem /usr/include -L/usr/lib/arm-linux-gnueabihf")
		case ArchArm64:
			image = runner.createContainerImage(arch, linuxOS, overrideDockerImage(flags.CommonFlags, linuxImageArm64))
			image.SetEnv("GOARCH", "arm64")
			image.SetEnv("CC", "zig cc -target aarch64-linux-gnu -isystem /usr/include -L/usr/lib/aarch64-linux-gnu")
			image.SetEnv("CXX", "zig c++ -target aarch64-linux-gnu -isystem /usr/include -L/usr/lib/aarch64-linux-gnu")
		}

		image.SetEnv("GOOS", "linux")

		cmd.Images = append(cmd.Images, image)
	}

	return nil
}
