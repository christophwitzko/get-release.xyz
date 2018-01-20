terraform {
  backend "gcs" {
    bucket = "get-release-tfstate"
    path   = "terraform.tfstate"
  }
}

variable "image" {
  type = "string"
}

variable "domain" {
  type = "string"
}

variable "ssh_user" {
  default = "ubuntu"
}

variable "private_key_path" {
  default = "~/.ssh/id_rsa"
}

variable "github_token" {
  type = "string"
}

variable "deploy_token" {
  type = "string"
}

resource "google_compute_instance" "default" {
  name         = "${element(split(".", "${var.domain}"), 0)}"
  machine_type = "f1-micro"
  zone         = "us-west1-a"
  tags         = ["http"]

  boot_disk {
    initialize_params {
      image = "${var.image}"
    }
  }

  network_interface {
    network = "default"

    access_config {
      # Ephemeral
    }
  }

  provisioner "remote-exec" {
    connection {
      type        = "ssh"
      user        = "${var.ssh_user}"
      private_key = "${file("${var.private_key_path}")}"
      agent       = false
    }

    inline = [
      "sudo docker run -d --name hub-webhook -e VIRTUAL_HOST=hub-webhook.${var.domain} -e LETSENCRYPT_HOST=hub-webhook.${var.domain} -e LETSENCRYPT_EMAIL=me@chw.io --restart=always -e DEFAULT_VHOST=${var.domain} -e DEFAULT_PARAMS='-e GITHUB_TOKEN=${var.github_token} -e LETSENCRYPT_EMAIL=me@chw.io' -e DEFAULT_TOKEN=${var.deploy_token} -v /var/run/docker.sock:/var/run/docker.sock:ro christophwitzko/docker-hub-webhook",
      "sudo docker run -d --name grd-server -e GITHUB_TOKEN=${var.github_token} -e LETSENCRYPT_EMAIL=me@chw.io --restart=always -e VIRTUAL_HOST=${var.domain} -e LETSENCRYPT_HOST=${var.domain} christophwitzko/grd-server",
    ]
  }
}

resource "google_compute_firewall" "default" {
  name    = "http-https"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["http"]
}

resource "google_dns_managed_zone" "default" {
  name     = "${replace("${var.domain}", ".", "-")}"
  dns_name = "${var.domain}."
}

resource "google_dns_record_set" "hub" {
  name         = "hub-webhook.${google_dns_managed_zone.default.dns_name}"
  managed_zone = "${google_dns_managed_zone.default.name}"
  type         = "A"
  ttl          = 300
  rrdatas      = ["${google_compute_instance.default.network_interface.0.access_config.0.assigned_nat_ip}"]
}

resource "google_dns_record_set" "default" {
  name         = "${google_dns_managed_zone.default.dns_name}"
  managed_zone = "${google_dns_managed_zone.default.name}"
  type         = "A"
  ttl          = 300
  rrdatas      = ["${google_compute_instance.default.network_interface.0.access_config.0.assigned_nat_ip}"]
}

output "ip" {
  value = "${google_compute_instance.default.network_interface.0.access_config.0.assigned_nat_ip}"
}

output "webhookurl" {
  value = "https://hub-webhook.${var.domain}/${var.deploy_token}"
}
