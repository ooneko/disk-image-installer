package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"diskimage-installer/cmd/options"
	"diskimage-installer/pkg/config"
	"diskimage-installer/pkg/installer"
	"diskimage-installer/pkg/utils"
)

func main() {
	CheckAllNeedCommandInstalled()
	options := options.Installer{}
	cmd := &cobra.Command{
		Use:   "disk-image-install",
		Short: "Install disk image into disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.Validate(); err != nil {
				return err
			}
			zapconfig := zap.NewProductionConfig()
			zapconfig.Level = zap.NewAtomicLevelAt(convertToZapLevel(options.LogLevel))
			logger, err := zapconfig.Build()
			zap.ReplaceGlobals(logger)
			if err != nil {
				return err
			}
			defer logger.Sync()

			if options.NodeConfig == "" {
				node := config.Node{}
				node.ImageInfo = &config.ImageInfo{
					Image: options.Image,
				}
				node.RootDevice = map[string]string{
					"name": options.RootDisk,
				}
				installer := installer.NewInstaller(node, logger)
				return installer.InstallOS()
			} else {
				// Do image install with raid config and generate configdrive
				data, err := ioutil.ReadFile(options.NodeConfig)
				if err != nil {
					logger.Sugar().Errorf("read file: %v", err)
					return err
				}
				var nodes []config.Node
				if err := yaml.Unmarshal(data, &nodes); err != nil {
					logger.Sugar().Errorf("yaml unmarshal: %v", err)
					return err
				}
				node, err := findLocalNode(nodes)
				if err != nil {
					logger.Sugar().Error(err)
					return err
				}
				installer := installer.NewInstaller(node, logger)
				return installer.InstallOS()
			}
		},
	}
	options.Addflags(cmd.Flags())
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func convertToZapLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		panic(fmt.Sprintf("unknown level %q", level))
	}

}

var allNeedCommand []string = []string{
	"mkisofs",
	"sgdisk",
	"qemu-img",
	"udevadm",
	"partprobe",
	"blkid",
	"wipefs",
	"parted",
	"blockdev",
	"efibootmgr",
}

func CheckAllNeedCommandInstalled() error {
	for _, command := range allNeedCommand {
		cmd := exec.Command("command", "-v", command)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s is required", command)
		}
	}
	return nil
}

func findLocalNode(nodes []config.Node) (config.Node, error) {
	out, err := utils.RunCommand("lshw", "-quiet", "-json")
	if err != nil {
		return config.Node{}, fmt.Errorf("lshw: %v", err)
	}
	data := map[string]interface{}{}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return config.Node{}, fmt.Errorf("json.Unmarshal: %v", err)
	}
	serial, ok := data["serial"]
	if !ok {
		return config.Node{}, fmt.Errorf("could not find serial from localhost")
	}
	for _, node := range nodes {
		if node.SerialNumber == serial.(string) {
			return node, nil
		}
	}
	return config.Node{}, fmt.Errorf("could not find node with match serial number with localhost from config")
}
