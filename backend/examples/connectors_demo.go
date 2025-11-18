package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"omniapi/internal/adapters/dummy"
	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	fmt.Println("🚀 OmniAPI Connectors Demo")
	fmt.Println("==========================")

	// 1. Registrar el conector dummy en el catálogo global
	fmt.Println("\n📋 Registrando conector dummy...")
	err := connectors.RegisterConnector(dummy.Registration)
	if err != nil {
		log.Printf("Warning: Failed to register dummy connector (may be already registered): %v", err)
	} else {
		fmt.Println("✅ Conector dummy registrado exitosamente")
	}

	// 2. Crear datos del dominio
	fmt.Println("\n🏗️  Creando datos del dominio...")
	tenantID := primitive.NewObjectID()
	typeID := primitive.NewObjectID()

	// Crear ConnectionInstance
	connectionInstance := domain.NewConnectionInstance(tenantID, typeID, "Demo Dummy Connection", "demo")
	connectionInstance.Config = map[string]interface{}{
		"farm_id": "demo-farm-001",
		"site_id": "demo-greenhouse-1",
		"cage_id": "demo-cage-A1",
	}

	// Crear ConnectorType
	connectorType := domain.NewConnectorType("dummy", "Demo Dummy Connector", "Demonstration connector", "1.0.0", "demo")
	connectorType.Capabilities = []domain.Capability{
		domain.CapabilityFeedingRead,
		domain.CapabilityBiometricRead,
		domain.CapabilityClimateRead,
	}

	fmt.Printf("   Tenant ID: %s\n", tenantID.Hex())
	fmt.Printf("   Connection: %s\n", connectionInstance.DisplayName)
	fmt.Printf("   Capabilities: %v\n", connectorType.Capabilities)

	// 3. Crear instancia del conector
	fmt.Println("\n⚙️  Creando instancia del conector...")
	instance, err := connectors.CreateConnectorInstance(connectionInstance, connectorType)
	if err != nil {
		log.Fatalf("❌ Error creando instancia: %v", err)
	}

	fmt.Printf("✅ Instancia creada: %s (tipo: %s)\n", instance.ID(), instance.Type())

	// 4. Configurar canal de eventos
	fmt.Println("\n📡 Configurando canal de eventos...")
	eventChan := make(chan connectors.CanonicalEvent, 50)
	instance.OnEvent(eventChan)

	// 5. Configurar filtros (opcional)
	filters := []connectors.EventFilter{
		{
			Capabilities: []domain.Capability{
				domain.CapabilityFeedingRead,
				domain.CapabilityBiometricRead,
				domain.CapabilityClimateRead,
			},
		},
	}

	err = instance.Subscribe(filters...)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to subscribe to filters: %v", err)
	}

	// 6. Iniciar el conector
	fmt.Println("\n▶️  Iniciando conector...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = instance.Start(ctx)
	if err != nil {
		log.Fatalf("❌ Error iniciando conector: %v", err)
	}

	fmt.Println("✅ Conector iniciado exitosamente")

	// 7. Monitorear eventos y salud
	fmt.Println("\n📊 Monitoreando eventos (10 segundos)...")
	fmt.Println("   Tipo     | Timestamp           | Source                    | Seq")
	fmt.Println("   ---------|---------------------|---------------------------|----")

	eventCount := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

monitorLoop:
	for {
		select {
		case event := <-eventChan:
			eventCount++
			timestamp := event.Envelope.Timestamp.Format("15:04:05")
			source := event.Envelope.Source
			if len(source) > 25 {
				source = source[:22] + "..."
			}

			fmt.Printf("   %-8s | %s | %-25s | %d\n",
				event.Kind,
				timestamp,
				source,
				event.Envelope.Sequence)

			// Mostrar payload del primer evento de cada tipo
			if eventCount <= 3 {
				fmt.Printf("            Payload: %s\n", string(event.Payload))
			}

		case <-ticker.C:
			// Mostrar estado de salud cada 2 segundos
			health := instance.Health()
			fmt.Printf("   [HEALTH] Status: %s, Events: %v, Uptime: %v\n",
				health.Status,
				health.Metrics["events_sent"],
				health.Uptime.Round(time.Second))

		case <-ctx.Done():
			break monitorLoop
		}
	}

	// 8. Detener el conector
	fmt.Println("\n⏹️  Deteniendo conector...")
	err = instance.Stop()
	if err != nil {
		log.Printf("⚠️  Warning: Error deteniendo conector: %v", err)
	} else {
		fmt.Println("✅ Conector detenido exitosamente")
	}

	// 9. Estadísticas finales
	fmt.Println("\n📈 Estadísticas finales:")
	health := instance.Health()
	fmt.Printf("   Total eventos procesados: %v\n", health.Metrics["events_sent"])
	fmt.Printf("   Estado final: %s\n", health.Status)
	fmt.Printf("   Errores: %d\n", health.ErrorCount)
	fmt.Printf("   Tiempo de ejecución: %v\n", health.Uptime.Round(time.Millisecond))

	// 10. Limpiar recursos
	fmt.Println("\n🧹 Limpiando recursos...")
	instanceID := connectionInstance.ID.Hex()
	err = connectors.GlobalCatalog.RemoveInstance(instanceID)
	if err != nil {
		log.Printf("⚠️  Warning: Error removiendo instancia del catálogo: %v", err)
	} else {
		fmt.Println("✅ Instancia removida del catálogo")
	}

	fmt.Println("\n🎉 Demo completada exitosamente!")
	fmt.Printf("📊 Resumen: %d eventos procesados en %v\n", eventCount, health.Uptime.Round(time.Second))
}
