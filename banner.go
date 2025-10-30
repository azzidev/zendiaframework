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
 ███████╗███████╗███╗   ██╗██████╗ ██╗ █████╗ 
 ╚══███╔╝██╔════╝████╗  ██║██╔══██╗██║██╔══██╗
   ███╔╝ █████╗  ██╔██╗ ██║██║  ██║██║███████║
  ███╔╝  ██╔══╝  ██║╚██╗██║██║  ██║██║██╔══██║
 ███████╗███████╗██║ ╚████║██████╔╝██║██║  ██║
 ╚══════╝╚══════╝╚═╝  ╚═══╝╚═════╝ ╚═╝╚═╝  ╚═╝`)
	fmt.Println("\033[0m") // Reset cor
	
	// Informações do framework
	fmt.Println("\033[1;32m🚀 ZendiaFramework - Go Multi-Tenant API Framework\033[0m")
	fmt.Println("\033[90m   Built with ❤️  for the Go community\033[0m")
	fmt.Println()
	
	// Informações da aplicação
	fmt.Printf("\033[1;34m📦 Application:\033[0m %s\n", config.AppName)
	fmt.Printf("\033[1;35m🔢 Version:\033[0m     %s\n", config.Version)
	fmt.Printf("\033[1;33m🌐 Port:\033[0m        %s\n", config.Port)
	fmt.Printf("\033[1;36m🔗 URL:\033[0m         http://localhost%s\n", config.Port)
	fmt.Println()
	
	// Features ativas
	fmt.Println("\033[1;32m✨ Active Features:\033[0m")
	fmt.Println("   🔐 Firebase Auth Integration")
	fmt.Println("   🏢 Multi-Tenant Support")
	fmt.Println("   📝 Automatic Audit Trail")
	fmt.Println("   🚀 Cache Layer (In-Memory)")
	fmt.Println("   📊 Health Checks & Monitoring")
	fmt.Println("   🔍 Request Tracing")
	fmt.Println("   📚 Auto Swagger Documentation")
	fmt.Println()
	
	if config.ShowRoutes {
		fmt.Println("\033[1;36m📋 Quick Links:\033[0m")
		fmt.Printf("   📖 Docs:    http://localhost%s/docs\n", config.Port)
		fmt.Printf("   🏥 Health:  http://localhost%s/health\n", config.Port)
		fmt.Printf("   📊 Metrics: http://localhost%s/public/metrics\n", config.Port)
		fmt.Println()
	}
	
	fmt.Println("\033[1;32m🎯 Ready to serve requests!\033[0m")
	fmt.Println("\033[90m" + strings.Repeat("─", 60) + "\033[0m")
	fmt.Println()
}