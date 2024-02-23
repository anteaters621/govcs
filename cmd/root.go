package cmd

import (
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"govcs/cmd/vcs"
	"log"
	"os"
	"strings"
)

const Version = "1.0.1"

const (
	ConfigNumber          = "number"
	ConfigColumns         = "columns"
	ConfigPadding         = "padding"
	ConfigThumbHeight     = "thumb_height"
	ConfigFontSize        = "font_size"
	ConfigFontFamily      = "font_family"
	ConfigVerbose         = "verbose"
	ConfigQuiet           = "quiet"
	ConfigFormat          = "format"
	ConfigBgColor         = "bg_color"
	ConfigFontColor       = "font_color"
	ConfigFFmpegPath      = "ffmpeg_path"
	ConfigFFprobe         = "ffprobe_path"
	ConfigOverwrite       = "overwrite"
	ConfigIgnoreExtension = "ignore_extension"
	ConfigJpegQuality     = "jpg_quality"
)

// ValidExtensions Don't try to vcs files that don't have one of these extensions
var ValidExtensions = [...]string{"3pg", "amv", "asf", "avi", "flv", "gif", "gifv", "m4v", "mkv", "mp4", "mpg", "mpeg", "mts", "ts", "ogv", "ogg", "rm", "rmvb", "vob", "webm", "wmv", "yuv"}

var (
	cfgFile string
)

var rootCmd = &cobra.Command{
	Use:     "govcs [flags] [files]",
	Short:   "Generate contact sheets for the given video files",
	Version: Version,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config := vcs.Config{
			Pics:            viper.GetInt(ConfigNumber),
			Columns:         viper.GetInt(ConfigColumns),
			ImagePadding:    viper.GetInt(ConfigPadding),
			ThumbHeight:     viper.GetInt(ConfigThumbHeight),
			FontSize:        viper.GetInt(ConfigFontSize),
			FontFamily:      viper.GetString(ConfigFontFamily),
			Verbose:         viper.GetBool(ConfigVerbose),
			Quiet:           viper.GetBool(ConfigQuiet),
			Format:          viper.GetString(ConfigFormat),
			JpegQuality:     viper.GetInt(ConfigJpegQuality),
			BgColor:         viper.GetString(ConfigBgColor),
			FontColor:       viper.GetString(ConfigFontColor),
			FFmpeg:          viper.GetString(ConfigFFmpegPath),
			FFprobe:         viper.GetString(ConfigFFprobe),
			IgnoreExtension: viper.GetBool(ConfigIgnoreExtension),
			Overwrite:       viper.GetBool(ConfigOverwrite),
		}

		if config.Verbose {
			fmt.Printf("Configuration:\n%+v\n", config)
		}

		if err := validate(config); err != nil {
			return err
		}

		for _, filename := range args {
			if !config.IgnoreExtension && !isVideoFile(filename) {
				continue
			}

			// Check if the file already exists and should not be overwritten
			if !config.Overwrite {
				targetFile := fmt.Sprintf("%s.%s", filename, config.Format)
				if _, err := os.Stat(targetFile); err == nil {
					if !config.Quiet {
						fmt.Printf("skipping %s (vcs already present)\n", filename)
						continue
					}
				}
			}

			if err := vcs.CreateVCS(filename, config); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().IntP(ConfigNumber, "n", 6, "number of still frames to take")
	_ = viper.BindPFlag(ConfigNumber, rootCmd.PersistentFlags().Lookup(ConfigNumber))

	rootCmd.PersistentFlags().IntP(ConfigColumns, "c", 6, "number of columns to align the stills")
	_ = viper.BindPFlag(ConfigColumns, rootCmd.PersistentFlags().Lookup(ConfigColumns))

	rootCmd.PersistentFlags().IntP(ConfigPadding, "", 8, "padding whitespace")
	_ = viper.BindPFlag(ConfigPadding, rootCmd.PersistentFlags().Lookup(ConfigPadding))

	rootCmd.PersistentFlags().IntP(ConfigThumbHeight, "", 200, "height of the generated stills")
	_ = viper.BindPFlag(ConfigThumbHeight, rootCmd.PersistentFlags().Lookup(ConfigThumbHeight))

	rootCmd.PersistentFlags().IntP(ConfigFontSize, "", 16, "font size for header")
	_ = viper.BindPFlag(ConfigFontSize, rootCmd.PersistentFlags().Lookup(ConfigFontSize))

	rootCmd.PersistentFlags().StringP(ConfigFormat, "", "jpg", "save file as jpg or png")
	_ = viper.BindPFlag(ConfigFormat, rootCmd.PersistentFlags().Lookup(ConfigFormat))

	rootCmd.PersistentFlags().StringP(ConfigFontFamily, "", "Deja Vu Sans", "font family to use")
	_ = viper.BindPFlag(ConfigFontFamily, rootCmd.PersistentFlags().Lookup(ConfigFontFamily))

	rootCmd.PersistentFlags().StringP(ConfigBgColor, "", "333333", "hex string for background color")
	_ = viper.BindPFlag(ConfigBgColor, rootCmd.PersistentFlags().Lookup(ConfigBgColor))

	rootCmd.PersistentFlags().StringP(ConfigFontColor, "", "DDDDDD", "hex string for header text color")
	_ = viper.BindPFlag(ConfigFontColor, rootCmd.PersistentFlags().Lookup(ConfigFontColor))

	rootCmd.PersistentFlags().BoolP(ConfigVerbose, "v", false, "show extra output")
	_ = viper.BindPFlag(ConfigVerbose, rootCmd.PersistentFlags().Lookup(ConfigVerbose))

	rootCmd.PersistentFlags().BoolP(ConfigQuiet, "q", false, "hide usual output")
	_ = viper.BindPFlag(ConfigQuiet, rootCmd.PersistentFlags().Lookup(ConfigQuiet))

	rootCmd.PersistentFlags().StringP(ConfigFFmpegPath, "", "ffmpeg", "path to the ffmpeg executable")
	_ = viper.BindPFlag(ConfigFFmpegPath, rootCmd.PersistentFlags().Lookup(ConfigFFmpegPath))

	rootCmd.PersistentFlags().StringP(ConfigFFprobe, "", "ffprobe", "path to the ffprobe executable")
	_ = viper.BindPFlag(ConfigFFprobe, rootCmd.PersistentFlags().Lookup(ConfigFFprobe))

	rootCmd.PersistentFlags().BoolP(ConfigOverwrite, "o", false, "overwrite existing files")
	_ = viper.BindPFlag(ConfigOverwrite, rootCmd.PersistentFlags().Lookup(ConfigOverwrite))

	rootCmd.PersistentFlags().BoolP(ConfigIgnoreExtension, "", false, "ignore file extension check to detect non-video formats")
	_ = viper.BindPFlag(ConfigIgnoreExtension, rootCmd.PersistentFlags().Lookup(ConfigIgnoreExtension))

	rootCmd.PersistentFlags().IntP(ConfigJpegQuality, "", 85, "quality of the jpeg when format is set to jpg")
	_ = viper.BindPFlag(ConfigJpegQuality, rootCmd.PersistentFlags().Lookup(ConfigJpegQuality))

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.govcs.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		// Search config in home directory with name ".govcs" without extension
		viper.AddConfigPath(home)
		viper.SetConfigName(".govcs")

		if err := viper.ReadInConfig(); err != nil {
			// can be dismissed
			if viper.GetInt(ConfigVerbose) > 0 {
				fmt.Printf("could not read config file %s/.govcs.ymaml\n", home)
				fmt.Printf("%s\n", err)
			}
		}
	}
}

func validate(config vcs.Config) error {
	if config.Pics < 1 {
		return errors.New("pics must be > 0")
	}
	if config.Columns < 1 {
		return errors.New("columns must be > 0")
	}
	if config.ImagePadding < 0 {
		return errors.New("image padding must be >= 0")
	}
	if config.ThumbHeight < 1 {
		return errors.New("thumb height must be > 0")
	}
	if config.FontSize < 1 {
		return errors.New("font size must be > 0")
	}
	if config.FontFamily == "" {
		return errors.New("font family not set")
	}
	if config.Format != "jpg" && config.Format != "png" {
		return errors.New("format must be jpg or png")
	}
	if config.FFmpeg == "" {
		return errors.New("ffmpeg not set")
	}
	if config.FFprobe == "" {
		return errors.New("ffprobe not set")
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		//log.Fatal(err)
	}
}

func isVideoFile(filepath string) bool {
	filepath = strings.ToLower(filepath)
	for _, extension := range ValidExtensions {
		if strings.HasSuffix(filepath, "."+extension) {
			return true
		}
	}
	return false
}
