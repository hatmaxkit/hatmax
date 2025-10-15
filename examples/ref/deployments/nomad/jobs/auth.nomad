job "auth" {
  datacenters = ["dc1"]
  type        = "service"

  group "auth" {
    count = 1

    network {
      port "http" {
        static = 8081
      }
    }

    service {
      name = "auth"
      port = "http"
      
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.auth.rule=PathPrefix(`/auth`)",
        "traefik.http.routers.auth.entrypoints=web",
        "auth","v1",
      ]

      check {
        type     = "http"
        path     = "/health"
        interval = "30s"
        timeout  = "5s"
      }
    }

    task "auth" {
      driver = "exec"

      config {
        command = "./auth"
        args    = ["--config", "/local/config.yaml"]
      }

      template {
        data = <<EOH
server:
  port: {{ env "NOMAD_PORT_http" }}
  host: "0.0.0.0"

database:
  type: "sqlite"
  dsn: "/alloc/data/auth.db"

logging:
  level: "info"
  format: "json"

observability:
  metrics:
    enabled: true
    endpoint: "/metrics"
  tracing:
    enabled: false

consul:
  address: "{{ env "CONSUL_HTTP_ADDR" | default "127.0.0.1:8500" }}"
  service_name: "auth"

EOH
        destination = "local/config.yaml"
        change_mode = "restart"
      }

      resources {
        cpu    = 256
        memory = 128
      }

      env {
        SERVICE_NAME = "auth"
        LOG_LEVEL    = "info"
        CONSUL_HTTP_ADDR = "{{ env "CONSUL_HTTP_ADDR" | default "127.0.0.1:8500" }}"
      }
    }
  }
}