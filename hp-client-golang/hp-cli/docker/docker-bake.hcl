variable "registry" {
  default = "docker.io"
}

variable "image_name" {
  default = "xrilang/hp-lite"
}

variable "image_tag" {
  default = "v26.1.14"
}

group "default" {
  targets = ["build"]
}

target "build" {
  context = ".."
  dockerfile = "docker/Dockerfile"
  tags = [
    "${registry}/${image_name}:latest",
    "${registry}/${image_name}:${image_tag}"
  ]
  push = true
}