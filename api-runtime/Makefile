default:
	        go build -o cb-spider
dyna plugin plug dynamic:
	        go build -tags dyna -o cb-spider-dyna
cc:
	        GOOS=linux GOARCH=arm go build -o cb-spider-arm
run:
	        ./cb-spider
clean:
	        rm -v cb-spider cb-spider-arm
