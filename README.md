# cardSlurp
This small utility aims to copy photos off of flash media as quickly
and as safely as possible.

cardSlurp uses the flags package, so it understands the -h option.

<pre>
$ ./cardSlurp -h
Usage of ./cardSlurp:
  -debugMode
        Print extra debug information.
  -mountDir string
        Directory where cards are mounted.
  -searchStr string
        String to distinguish cards from other mounted media in mountDir.
  -targetDir string
        Target directory for the copied files.
</pre>

Here is a usage example.  (Yes, the author shoots Canon, for what it
is worth.)

<pre>
./cardSlurp -mountDir /media/USER -searchStr EOS_DIG -targetDir /tmp/thingTwo
</pre>

About the author: Patrick works as a devops developer for a financial
services company, and he is a passionate photographer.