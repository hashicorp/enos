resource "random_string" "random" {
  length  = 8
  special = true
  number  = true // deprecated, should be numeric, so we'll generate a warning
}
