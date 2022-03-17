variable "foo" {
  description = "a useful variable"
  sensitive   = true
  type = object({
    value = number
  })
  default = {
    value = 1
  }
}

scenario "another" {
}
