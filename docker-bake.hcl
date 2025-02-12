target "httpserver" {
  context = "."
  dockerfile = "httpserver.Dockerfile"
  tags = ["ghcr.io/ernado/ki/httpserver:latest"]
}
