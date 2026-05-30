package zendia

import (
	"fmt"
	"strings"
)

// BannerConfig configuração do banner
type BannerConfig struct {
	AppName    string
	Version    string
	Port       string
	ShowRoutes bool
}

// ShowBanner exibe o banner do framework
func (z *Zendia) ShowBanner(config BannerConfig) {
	// ASCII Art do ZendiaFramework
	fmt.Println("\033[36m") // Cor ciano
	fmt.Println(`
                                   █████ ███           
                                  ░░███ ░░░            
 █████████  ██████  ████████    ███████ ████   ██████  
░█░░░░███  ███░░███░░███░░███  ███░░███░░███  ░░░░░███ 
░   ███░  ░███████  ░███ ░███ ░███ ░███ ░███   ███████  v2
  ███░   █░███░░░   ░███ ░███ ░███ ░███ ░███  ███░░███ 
 █████████░░██████  ████ █████░░█████████████░░████████
░░░░░░░░░  ░░░░░░  ░░░░ ░░░░░  ░░░░░░░░░░░░░  ░░░░░░░░ `)
	fmt.Println("\033[0m") // Reset cor

	// Informações do framework
	fmt.Println("\033[1;32m🚀 ZendiaFramework - Framework Go Multi-Tenant para APIs\033[0m")
	fmt.Println("\033[90m   Feito com ❤️  para a comunidade Go brasileira\033[0m")
	fmt.Println()

	// Informações da aplicação
	fmt.Printf("\033[1;34m📦 Aplicação:\033[0m %s\n", config.AppName)
	fmt.Printf("\033[1;35m🔢 Versão:\033[0m    %s\n", config.Version)
	fmt.Printf("\033[1;33m🌐 Porta:\033[0m     %s\n", config.Port)
	fmt.Printf("\033[1;36m🔗 URL:\033[0m       http://localhost%s\n", config.Port)
	fmt.Println()

	if config.ShowRoutes {
		routes := z.engine.Routes()
		fmt.Printf("\033[1;36m📋 Rotas registradas (%d):\033[0m\n", len(routes))
		for _, r := range routes {
			fmt.Printf("   \033[1;33m%-7s\033[0m %s\n", r.Method, r.Path)
		}
		fmt.Println()
	}

	fmt.Println("\033[1;32m🎯 Pronto para servir requisições!\033[0m")
	fmt.Println("\033[90m" + strings.Repeat("─", 60) + "\033[0m")
	fmt.Println()
}
