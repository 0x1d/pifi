build:
	env GOOS=linux GOARCH=arm64 go build -o pifi main.go

upload:
	scp pifi pi@rpi-40ac:/home/pi/pifi
