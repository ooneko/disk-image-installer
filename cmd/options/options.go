package options

import (
	"fmt"

	"github.com/spf13/pflag"
)

type Installer struct {
	LogLevel   string
	NodeConfig string
	Image      string
	RootDisk   string
}

func (i *Installer) Addflags(fs *pflag.FlagSet) {
	fs.StringVar(&i.LogLevel, "log-level", "info", "available level: debug, info, warn, error, dpanic, panic, fatal")
	fs.StringVar(&i.NodeConfig, "nodeconfig", "", "path of nodeconfig")
	fs.StringVar(&i.Image, "image", "", "image file to write to disk")
	fs.StringVar(&i.RootDisk, "root-disk", "/dev/sda", "root disk to written image")
}

func (i *Installer) Validate() error {
	if i.Image == "" && i.NodeConfig == "" {
		return fmt.Errorf("neither image or nodeconfig are not specified")
	}
	return nil
}
