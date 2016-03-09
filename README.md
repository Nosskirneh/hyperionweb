# hyperionweb

> Web server written in GO for advanced management of hyperion.

Keeps track of last color, whether there is an efect running or if plex is using the leds. Also turns on the leds when you're coming home and turn them off when you disconnect from wifi.

![page](/screenshots/page.png)

## Installation
1. [Download golang](https://golang.org/dl/).

2. Configure the parameters located within the const block in hyperionweb.go

3. Add your local id_rsa.pub to the authorized_keys on the remote machine to be able to login without password. Tip: use [ssh-copy-id](http://linux.die.net/man/1/ssh-copy-id).

4. Build the package (go build hyperionweb.go) and run it.

5. If using Arch Linux, copy the /arch/hyperionweb.service to /etc/systemd/system/ and use it as a normal systemd service.

## Acknowledgements 
This is a fork of [Bryal](https://github.com/Bryal)'s unfinished hyperionweb.