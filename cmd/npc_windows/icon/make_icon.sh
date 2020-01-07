#/bin/sh

if [ -z "$GOPATH" ]; then
    echo GOPATH environment variable not set
    exit
fi

if [ ! -e "$GOPATH/bin/2goarray" ]; then
    echo "Installing 2goarray..."
    go get github.com/cratonica/2goarray
    if [ $? -ne 0 ]; then
        echo Failure executing go get github.com/cratonica/2goarray
        exit
    fi
fi

if [ -z "$1" ]; then
    echo Please specify a ICO file
    exit
fi

if [ ! -f "$1" ]; then
    echo $1 is not a valid file
    exit
fi    

echo Creating iconwin.go
echo "//+build windows" > iconwin.go
echo >> iconwin.go
cat "$1" | $GOPATH/bin/2goarray Data icon >> iconwin.go
if [ $? -ne 0 ]; then
    echo Failure generating iconwin.go
    exit
fi

echo Finished
