#! /bin/sh

# pass user@host as a parameter to automatically install on a remote host
# note: requires ssh access to the remote host for this to work
SERVER=$1

# settings
INSTALL=sandbox
SOURCE=cmd
MAIN=main.go
BIN=/usr/local/bin

echo "\nCREATE $INSTALL"
mkdir $INSTALL

echo "BUILD kvs"
GOOS=linux GOARCH=amd64 go build \
-trimpath \
-ldflags "-s -w" \
-o $INSTALL/kvs $SOURCE/kvs/$MAIN

echo "BUILD kvs-keon"
GOOS=linux GOARCH=amd64 go build \
-trimpath \
-ldflags "-s -w" \
-o $INSTALL/kvs-keon $SOURCE/kvs-keon/$MAIN

echo "BUILD kvs-keva"
GOOS=linux GOARCH=amd64 go build \
-trimpath \
-ldflags "-s -w" \
-o $INSTALL/kvs-keva $SOURCE/kvs-keva/$MAIN

# automated remote server installation
if [ "$SERVER" ]; then 
    
    echo "TRANSMIT to $SERVER"
    scp $INSTALL/kvs $SERVER:
    scp $INSTALL/kvs-keon $SERVER:
    scp $INSTALL/kvs-keva $SERVER:
    SSH $SERVER "sudo mv kvs* $BIN"
    echo "INSTALLED"
    
    echo "REMOVE $INSTALL"
    rm -fr $INSTALL
fi

echo "bye...\n"