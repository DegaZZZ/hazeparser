# HazeParser


Parser for Deadlock replay files written in Go. Early stages of development.

### Usage

1. Compile using ```go build```

2. Parse using ```.\hazeparser -f yourdemo.dem``` in console.

In the output you will get the `output.txt` with all of the messages inside demo as well as file `match_data.json`. JSON file has all the data from `CCitadelUserMsg_PostMatchDetails` (**ubit = 316**).

### Flags

1. `-f` sets a demo file
2. `-qm` turns on quick mode that significantly increases (10x boost) by not outputing insides of Packets into output.txt.  
