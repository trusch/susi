#!/bin/bash
#
# This parses all jade files and converts the include to makefile rules;
# These rules can be integrated into the jade makefile;
#

currentDir=$(pwd)

for f in templates/**/*.jade; do
	dir=$(dirname $f)
	includes=$(cat $f|grep include|sed s/include//g)
	echo -n "$(dirname $f)/$(basename $f .jade).js: $f "
	for include in $includes; do
		need=$(realpath $dir/$include.js 2>/dev/null)
		need=${need#$currentDir/}
		if test $? = 0; then
			echo -n "$need "
		fi
	done
	echo -en "\n"
done

exit 0
