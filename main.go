package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/spf13/viper"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

type Config struct {
	Profiles map[string]ProfileConfig `mapstructure:"profiles"`
}

type ProfileConfig struct {
	ECR    ECRConfig    `mapstructure:"ecr"`
	Docker DockerConfig `mapstructure:"docker"`
}

type ECRConfig struct {
	Region     string `mapstructure:"region"`
	AccountID  string `mapstructure:"account_id"`
	Repository string `mapstructure:"repository"`
	ImageTag   string `mapstructure:"image_tag"`
}

type DockerConfig struct {
	ImageName string `mapstructure:"image_name"`
}

type ECR struct {
	Config *ProfileConfig
}

func main() {
	configPath := flag.String("config", "deploy.yml", "Path to the configuration YAML file")
	profile := flag.String("profile", "dev", "Configuration profile to use (e.g., dev, prod)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s -config deploy.yml -profile dev [opciones]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Println(ColorRed + "Error loading configuration: " + err.Error() + ColorReset)
		os.Exit(1)
	}

	profileConfig, exists := config.Profiles[*profile]
	if !exists {
		fmt.Printf(ColorRed+"Profile '%s' not found in configuration"+ColorReset+"\n", *profile)
		os.Exit(1)
	}

	fmt.Printf(ColorYellow+"Loaded Configuration for profile '%s': %+v"+ColorReset+"\n", *profile, profileConfig)

	if err := validateConfig(&profileConfig); err != nil {
		fmt.Println(ColorRed + "Invalid configuration: " + err.Error() + ColorReset)
		os.Exit(1)
	}

	ecr := &ECR{Config: &profileConfig}

	if err := ecr.authenticate(); err != nil {
		fmt.Println(ColorRed + "Authentication failed: " + err.Error() + ColorReset)
		os.Exit(1)
	}

	if err := ecr.build(); err != nil {
		fmt.Println(ColorRed + "Build failed: " + err.Error() + ColorReset)
		os.Exit(1)
	}

	if err := ecr.tag(); err != nil {
		fmt.Println(ColorRed + "Tag failed: " + err.Error() + ColorReset)
		os.Exit(1)
	}

	if err := ecr.push(); err != nil {
		fmt.Println(ColorRed + "Push failed: " + err.Error() + ColorReset)
		os.Exit(1)
	}

	fmt.Println(ColorGreen + "Container built and pushed to ECR" + ColorReset)
}

func loadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	viper.SetDefault("profiles.dev.ecr.region", "us-east-1")
	viper.SetDefault("profiles.dev.ecr.image_tag", "latest")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error leyendo el archivo de configuración: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error parseando la configuración: %w", err)
	}

	return &config, nil
}

func validateConfig(config *ProfileConfig) error {
	if config.ECR.Region == "" {
		return fmt.Errorf("ecr.region is required")
	}
	if config.ECR.AccountID == "" {
		return fmt.Errorf("ecr.account_id is required")
	}
	matched, err := regexp.MatchString(`^\d{12}$`, config.ECR.AccountID)
	if err != nil || !matched {
		return fmt.Errorf("ecr.account_id debe ser una cadena de 12 dígitos")
	}
	if config.ECR.Repository == "" {
		return fmt.Errorf("ecr.repository is required")
	}
	if config.Docker.ImageName == "" {
		return fmt.Errorf("docker.image_name is required")
	}
	return nil
}

func (ecr *ECR) authenticate() error {
	fmt.Println(ColorCyan + "Authenticating Docker with ECR" + ColorReset)
	ecrRepo := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", ecr.Config.ECR.AccountID, ecr.Config.ECR.Region)
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("aws ecr get-login-password --region %s | docker login --username AWS --password-stdin %s",
			ecr.Config.ECR.Region,
			ecrRepo,
		),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error durante la autenticación con ECR: %w", err)
	}
	return nil
}

func (ecr *ECR) build() error {
	fmt.Println(ColorCyan + "Building container" + ColorReset)
	build := exec.Command("docker", "build", "-t", ecr.Config.Docker.ImageName, ".")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("error al construir la imagen Docker: %w", err)
	}
	return nil
}

func (ecr *ECR) tag() error {
	fmt.Println(ColorYellow + "Tagging container" + ColorReset)
	localImage := fmt.Sprintf("%s:%s", ecr.Config.Docker.ImageName, ecr.Config.ECR.ImageTag)
	ecrImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
		ecr.Config.ECR.AccountID,
		ecr.Config.ECR.Region,
		ecr.Config.ECR.Repository,
		ecr.Config.ECR.ImageTag,
	)
	tag := exec.Command("docker", "tag", localImage, ecrImage)
	tag.Stdout = os.Stdout
	tag.Stderr = os.Stderr
	if err := tag.Run(); err != nil {
		return fmt.Errorf("error al etiquetar la imagen Docker: %w", err)
	}
	return nil
}

func (ecr *ECR) push() error {
	fmt.Println(ColorCyan + "Pushing container" + ColorReset)
	ecrImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
		ecr.Config.ECR.AccountID,
		ecr.Config.ECR.Region,
		ecr.Config.ECR.Repository,
		ecr.Config.ECR.ImageTag,
	)
	push := exec.Command("docker", "push", ecrImage)
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("error al empujar la imagen Docker: %w", err)
	}
	return nil
}
