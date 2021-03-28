# labns

A basic nameserver implementation for your home or lab environment.

## features
- 2 user defined upstream nameservers (primary + secondary)
- user defined, A, AAAA and CNAME records
- see `labns.json` for an example configuration file

## installation

### systemd
- requires a linux distro with systemd and golang 1.15+ in the system path
- run the `systemd-install.sh` script as a super user or root
- enable labns to start on boot (if desired): `sudo systemctl enable labns.service`
- start labns as superuser: `sudo systemctl start labns.service`
- check the status of labns: `sudo systemctl status labns`

### docker
- build the image from the dockerfile: `docker build -t labns:prod .`
- run the image, exposing UDP port 53 and mount the configuration file to the container at runtime e.g.
```
docker run --name labns -p 53:53/udp -v /path/to/config.json:/dist/config.json labns:prod
```

## environment
\
labns supports a number of configuration parameters parsed as environment variables:
\
`LABNS_CONFIG_PATH`: an absolute path to the JSON configuration file (defaults to /etc/labns/labns.json)
\
`LABNS_DNS_SERVICE_PORT`: specify a non standard port to start the UDP listener on (defaults to 53)
\
`LABNS_LOG_PATH`: specify a log file to redirect stdout and stderr into (note this will prevent the service from logging to stdout)



## Notes

Note that in order for clients to use your labns host as a nameserver you will need to open port 53 to incoming UDP traffic in your system firewall with a tool such as iptables or firewalld.

Stay tuned for more features!
