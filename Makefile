clean:
	go clean -testcache && rm -r ./bin

update:
	go get -u ./... && go mod tidy

buildfoma_de:
	cd src && \
	foma -e "source de/tokenizer.xfst" \
	-e "save stack ../testdata/tokenizer.fst" -q -s && \
	cd ..

buildfoma_en:
	cd src && \
	foma -e "source en/tokenizer.xfst" \
	-e "save stack ../testdata/tokenizer_en.fst" -q -s && \
	cd ..

buildmatok: buildfoma build
	./bin/datok convert -i ./testdata/tokenizer.fst -o ./testdata/tokenizer.matok

buildmatok_en: buildfoma_en build
	./bin/datok convert -i ./testdata/tokenizer_en.fst -o ./testdata/tokenizer_en.matok

builddatok: buildfoma build
	./bin/datok convert -i ./testdata/tokenizer.fst -o ./testdata/tokenizer.datok -d

test:
	go test ./...

build:
	go build -v -o ./bin/datok ./cmd/datok.go

benchmark:
	go test -bench=. -test.benchmem
