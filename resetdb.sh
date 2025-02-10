#! /bin/bash

if [ -f ./.env ]; then 
	source ./.env

	cd ./sql/schema
	echo -e '\n === dropping all tables in database'
	goose $GOOSE_DRIVER $GOOSE_DBSTRING down
	echo -e '\n === migrating to latest'
	goose $GOOSE_DRIVER $GOOSE_DBSTRING up

else
	echo -e '\n === WARNING: ".\.env" file not found.\n === Did not perform database reset.\n'
fi
