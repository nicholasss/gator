#! /bin/bash

if [ -f ./.env ]; then 
	source ./.env

	cd ./sql/schema
	echo -e '\n === dropping all tables in database'
	goose down $GOOSE_DRIVER $GOOSE_DBSTRING
	echo -e '\n === migrating to latest'
	goose up $GOOSE_DRIVER $GOOSE_DBSTRING

else
	echo -e '\n === WARNING: ".\.env" file not found.\n === Did not perform database reset.\n'
fi
