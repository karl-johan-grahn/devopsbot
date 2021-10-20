#!/bin/bash
if (git describe --abbrev=0 --exact-match &>/dev/null); then
  # If we are on a tagged commit, use that tag as version
  git describe --abbrev=0 --exact-match | sed 's/v\(.*\)/\1/'
else
  # Otherwise get the latest tagged commit
  tag=$(git rev-list --tags --max-count=1 2>/dev/null)
  if [ "$tag" == "" ]; then
    version="0.0.0"
  else
    version=$(git describe --abbrev=0 --tags $tag 2>/dev/null | sed 's/v\(.*\)/\1/')
  fi
  # This is in development
  echo "${version}-dev"
fi
