#!/bin/bash
npm outdated | awk 'NR>1 {print $1"@"$4}' | xargs npm install