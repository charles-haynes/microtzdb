# microtzdb

A tiny version of the timezone database intended for use on embedded systems. It contains a map from hashed versions of the timezone names to indexes in a lookuptable of POSIX timezone strings. It's useful when you know your timezone name ("Olson timzone") but need a POSIX timezone string to set the `TZ` environment variable.

## Usage

Include this code in your app. It provides a function `const char *getPosixTZforOlson(const char *olson, char *buf, size_t buflen)` that takes the name of a timezone (like "Australia/Melbourne") and returns the POSIX timezone string for that timezone (like "AEST-10AEDT,M10.1.0,M4.1.0/3") that can be used to set the `TZ` environment variable that controls timzone conversion on POSIX
compliant systems (and things like Arduino and ESP32.)

`getPosixTZforOlson` does no error checking. If you pass it something that isn't an Olson timezone name the results are undefined. It might return `nullptr` or it could return a random timezone string. Caveat emptor.

## Performance

The library takes about 4k bytes on an ESP32. It's pretty performant, each lookup should take about eight compares to do the lookup. It might be possible to compress things a bit further using a perfect hash, it's unlikely to ever be much smaller than 2k bytes.

## Alternatives

Rop Gonggrijp provides a free UDP based server [timzoned.rop.nl](https://github.com/ropg/ezTime#timezonedropnl) that will convert Olson to POSIX.

## Updates

The updates directory contains a script to fetch updated versions of the database and install them and a go program to generate the `microtzdb.c` file. Look at the script `update.sh` you will need `curl`, `tar`, and the C and Go toolchains installed.