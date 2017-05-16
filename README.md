# Dockerized SHASum Cacher

This tool uses Go, SSL, and Redis to calculate SHA256 sums, store them, and
return them to the user for lookup later.

The encryption keys are not bundled with the tool for security reasons. Users
can create a keypair in this directory by running:

`openssl req -x509 -sha256 -nodes -newkey rsa:2048 -days 365 -keyout localhost.key -out localhost.crt -subj '/CN=localhost'`

With those keys created, the SHASum Cacher container can be launched with

`docker-compose up`

By default it will listen on port 5000, but that is a command line flag that
can be modified by editing the "command" line in `docker-compose.yml`. Were the
caching HTTPS server to crash for any reason, Docker would restart it.

Enjoy!
