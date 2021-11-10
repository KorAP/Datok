clean:
	go clean -testcache && rm -r ./bin

buildfoma:
	cd src && \
	foma -e "source tokenizer.xfst" \
	-e "save stack ../testdata/tokenizer.fst" -q -s && \
	cd ..

buildmatok: buildfoma build
	./bin/datok convert -i ./testdata/tokenizer.fst -o ./testdata/tokenizer.matok

builddatok: buildfoma build
	./bin/datok convert -i ./testdata/tokenizer.fst -o ./testdata/tokenizer.datok -d

test:
	go test ./...

build:
	go build -v -o ./bin/datok ./cmd/datok.go

benchmark:
	go test -bench=. -test.benchmem
