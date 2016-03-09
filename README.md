# hyperionweb

> Web server written in GO for advanced management of hyperion.

Keeps track of last color, whether there is an efect running or if plex is using the leds. Also turns on the leds when you're coming home and turn them of when you disconnect from wifi.

![page](/screenshots/page.png)

## Installation
1. [Download golang](https://golang.org/dl/).

2. Configure the parameters located within the const block in hyperionweb.go

3. Add your local id_rsa.pub to the authorized_keys on the remote machine to be able to login without password. Tip: use [ssh-copy-id](http://linux.die.net/man/1/ssh-copy-id).

4. Build the package (go build hyperionweb.go) and run it.

## Acknowledgements 
This is a fork of [Bryal](https://github.com/Bryal)'s unfinished hyperionweb.