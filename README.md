# Haze Parser


Parser for Deadlock replay files written in Go. Early stages of development.

### Usage

1. Compile using ```go build```

2. Parse using ```.\hazeparser -f yourdemo.dem``` in console.

In the output you will get the `output.txt` with all of the messages inside demo as well as file `match_data.json`. JSON file has all the data from `CCitadelUserMsg_PostMatchDetails` (**ubit = 316**). Don't forget to clean delete output.txt before parsing another demo (will fix it later).

### Flags

1. `-f` sets a demo file
2. `-o` saves parsing log into `output.txt` for debugging purposes

### TODO



- [X] Implement basic parser
- [X] Remove redundant -qm flag by making parser faster
- [X] Get PostMatchDetails data in JSON format
- [ ] Fix a need to clean output.txt before parsing
- [ ] Further QOL improvements
- [ ] Implement handling of more ubits
- [ ] Add more ubits into docs
- [ ] Implement Python library for parsing
- [ ] Implement string tables
- [ ] Implement additional Python module for analytics

