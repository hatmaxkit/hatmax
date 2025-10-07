job "todo" {
  datacenters = ["dc1"]
  type        = "service"

  group "todo" {
    count = 1

    network {
      port "http" {
        static = 8080
      }
    }

    service {
      name = "todo"
      port = "http"
      
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.todo.rule=PathPrefix(`/todo`)",
        "traefik.http.routers.todo.entrypoints=web",
        "api","v1",
      ]

      check {
        type     = "http"
        path     = "/health"
        interval = "30s"
        timeout  = "5s"
      }
    }

    task "todo" {
      driver = "exec"

      config {
        command = "./todo"
        args    = ["--config", "/local/config.yaml"]
      }

      template {
        data = <<EOH
server:
  port: {{ env "NOMAD_PORT_http" }}
  host: "0.0.0.0"

database:
  type: "sqlite"
  dsn: "/alloc/data/todo.db"

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
  service_name: "todo"

EOH
        destination = "local/config.yaml"
        change_mode = "restart"
      }

      resources {
        cpu    = 256
        memory = 128
      }

      env {
        SERVICE_NAME = "todo"
        LOG_LEVEL    = "info"
        CONSUL_HTTP_ADDR = "{{ env "CONSUL_HTTP_ADDR" | default "127.0.0.1:8500" }}"
      }
    }
  }
}