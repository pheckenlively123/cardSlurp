# cardSlurp

## Introduction

This repo contains two small utilities that the author finds helpful
for his photography.

## cardslurp

This utility copies all the files off of photo cards to the specified
target directory.  As each file is copied, `cardslurp` verifies
that the source and target files are byte for byte the same.  If
there are name conflicts in the target directory, `cardslurp`
automatically adjusts the target file name to avoid overwriting
files that are already there.

PRO TIP: If you shoot with multiple cameras, like the author of this tool,
adjust the names of the files generated by each camera, so they can never
conflict.  Most cameras support this feature, and Canon EOS cameras definitely
do.  For example, one of my cameras generates files named `PAH_####.CR2` and the
other generates files named `PBH_####.CR2`.

`cardslurp` uses the `flag` package, so it understands the `-h` option.

```
patrickheckenlively@Patricks-Mac-Studio:~$ ~/myBin/cardslurp -h
Usage of /Users/patrickheckenlively/myBin/cardslurp:
  -debugMode
    	Print extra debug information.
  -maxretries uint
    	Max number of retry attempts. (default 5)
  -mountlist string
    	Comma delimited list of mounted cards.
  -targetdir string
    	Target directory for the copied files.
  -verifychunksize uint
    	Size of the verify chunks (default 16384)
  -verifypasses uint
    	Number of file verify test passes (default 3)
  -workerpool uint
    	Size of the worker pool (default 4)
patrickheckenlively@Patricks-Mac-Studio:~$ 
```

Most of these command line arguments can be ignored in common
practice.  Examples of typical usage are provied below for Linux,
Mac, and Windows.

### Linux

```
./cardslurp -mountlist="/media/someuser/EOS_DIGITAL,/media/someuser/EOS_DIGITAL1" -targetdir="/somewhere"
```

### Windows

```
.\cardslurp -mountlist="I:,F:" -targetdir="c:\somewhere"
```

### MacOS

```
./cardslurp -mountlist="/Volumes/EOS_DIGITAL,/Volumes/EOS_DIGITIAL 1" -targetdir="/somewhere"
```

### Installation

```
cd cmd/cardslurp
make TARGET
```

In the commands above, replace TARGET with one of the options below, based on
the architecture of the computer where you wish to run cardslurp.

* mac_arm64
* mac_amd64
* linux_amd64
* win_amd64

Copy the resulting binary to the desired location, once it is built.

## xmpsafecopy

This utility is probably only of interest, if your photo workflow
is similar to mine.  I like to keep my images on an Samba based
file share.  However, to speed up the culling process, I sometimes
copy the image directory to local disk on my MacStudio.  I have
PhotoMechanic and Lightroom configured to write metadata to `.xmp`
side cart files.  This allows me to use PhotoMechanic to cull the
photoshoot directory.  I then copy the `.xmp` files back to the
corresponding directory on the Samba file share.  (Lightroom only
knows about the directory on the Samba file share.)  Then I tell
Lightroom to import metadata from the images.

I wrote this utility, so I could programatically ensure that we are
only moving `.xmp` files associated with the same photo shoot.

`xmpsafecopy` uses the `flag` package, so it understands the `-h` command line option.

```
~/myBin/xmpsafecopy -h
Usage of /Users/patrickheckenlively/myBin/xmpsafecopy:
  -extension string
    	File extension (default "xmp")
  -memorex
    	Is it live, or is it memorex (default true)
  -source string
    	Source directory
  -target string
    	Target directory
```

### Example Usage

```
~/myBin/xmpsafecopy -source="/somesource" -target="/sometarget" -memorex=false
```

### Installation

Run the commands below on machine with the Go SDK and Make installed.

```
cd cmd/xmpsafecopy
make TARGET
```

Acceptable values for TARGET are:

* mac_arm64
* mac_amd64
* win_amd64
* linux_amd64

Copy the resulting binary to the desired location, once it is built.

# The Author

Patrick is an SRE with over 20 years of industry experience, and
he is a passionate photographer.