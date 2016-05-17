# go-butler
A mumble bot based on the [Gumble libary](https://github.com/layeh/gumble/)

[![Build Status](https://drone.io/github.com/njdart/go-butler/status.png)](https://drone.io/github.com/njdart/go-butler/latest)

##To run
**(Requires golang)**
- ```git clone https://github.com/njdart/go-butler.git```
- ```cd go-butler```
- ```cp ./config.json.example ./config.json```
- Edit config as necessary
- ```go run go-butler.go```

## Features
- [x] Load from config file
- [x] logging
- [ ] Tests
- [ ] check ACL's so that cmds can be made admin only
- [ ] check status of Seam and other services
- [ ] format source connect cmds to button
- [ ] Be able to talk back to users