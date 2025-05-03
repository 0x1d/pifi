build:
	env GOOS=linux GOARCH=${ARCH} go build -o pifi main.go

upload:
	scp pifi pi@rpi-40ac:/home/pi/pifi
