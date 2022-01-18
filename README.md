# paste-netcat

Allows you to upload content to [paste](https://github.com/lucko/paste) (and [bytebin](https://github.com/lucko/bytebin)) using netcat.

## Example

#### Start an instance of paste-netcat with Docker
```shell
> git clone https://github.com/lucko/paste-netcat
> cd paste-netcat
> vi docker-compose.yml # edit the config
> docker compose up -d
```

#### Upload content
```shell
# pipe in some output from any command
> echo "Hello world" | nc localhost 3000
https://pastes.dev/aaaaa

# upload the contents of a file
> cat some_file.txt | nc localhost 3000
https://pastes.dev/bbbbb

# read back the contents
> curl https://pastes.dev/aaaaa
Hello world
```

### About
Written in Go, licensed MIT, have fun. :)