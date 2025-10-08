package zendia

import (
	"log"
	"strings"
)

// BannerConfig configuração do banner
type BannerConfig struct {
	AppName    string
	Version    string
	Port       string
	ShowRoutes bool
}

// ShowBanner exibe banner automático com rotas registradas
func (z *Zendia) ShowBanner(config BannerConfig) {
	if config.AppName == "" {
		config.AppName = "ZendiaFramework App"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.Port == "" {
		config.Port = "8080"
	}

	// Banner principal
	log.Printf("🚀 %s v%s running on :%s", config.AppName, config.Version, config.Port)

	// Info de autenticação
	if z.authConfig != nil {
		log.Println("🔐 Firebase Authentication enabled")
		if len(z.authConfig.PublicRoutes) > 0 {
			log.Printf("📋 Public routes: %s", strings.Join(z.authConfig.PublicRoutes, ", "))
		}
		log.Println("🔗 Use: Authorization: Bearer <firebase-token>")
	} else {
		log.Println("📋 Use headers: X-Tenant-ID and X-User-ID")
	}

	// Rotas automáticas
	if config.ShowRoutes {
		z.showRegisteredRoutes()
	}

	log.Println("✅ Server ready!")
}

// showRegisteredRoutes mostra rotas registradas automaticamente
func (z *Zendia) showRegisteredRoutes() {
	routes := z.engine.Routes()

	log.Println("🔗 Registered endpoints:")

	// Agrupa por método
	methodGroups := make(map[string][]string)
	for _, route := range routes {
		if route.Path != "" {
			methodGroups[route.Method] = append(methodGroups[route.Method], route.Path)
		}
	}

	// Exibe organizadamente
	for method, paths := range methodGroups {
		for _, path := range paths {
			log.Printf("  %s %s", method, path)
		}
	}
}
