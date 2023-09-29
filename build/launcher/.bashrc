#!/bin/bash

# lovingly copied from @loudambiance
# https://gist.github.com/loudambiance/a41b42a4295bce6e7304

WHITE='\[\033[1;37m\]'
LIGHTRED='\[\033[1;31m\]'
LIGHTGREEN='\[\033[1;32m\]'
LIGHTBLUE='\[\033[1;34m\]'
DEFAULT='\[\033[0m\]'

cLINES=$WHITE #Lines and Arrow
cBRACKETS=$WHITE # Brackets around each data item
cERROR=$LIGHTRED # Error block when previous command did not return 0
cSUCCESS=$LIGHTGREEN  # When last command ran successfully and return 0
cHST=$LIGHTGREEN # Color of hostname
cPWD=$LIGHTBLUE # Color of current directory
cCMD=$DEFAULT # Color of the command you type

function promptcmd()
{
        PREVRET=$?

        # new line to clear space from previous command
        PS1="\n"

        # prev cmd error
        if [ $PREVRET -ne 0 ] ; then
                PS1="${PS1}${cBRACKETS}[${cERROR}x${cBRACKETS}]${cLINES}\342\224\200"
        else
                PS1="${PS1}${cBRACKETS}[${cSUCCESS}*${cBRACKETS}]${cLINES}\342\224\200"
        fi

        # user
        PS1="${PS1}${cBRACKETS}[${cHST}\h${cBRACKETS}]${cLINES}\342\224\200"

        # dir
        PS1="${PS1}[${cPWD}\w${cBRACKETS}]"

        # second line
        PS1="${PS1}\n${cLINES}\342\224\224\342\224\200\342\224\200> ${cCMD}"
}

function load_prompt () {
    PROMPT_COMMAND=promptcmd

    export PS1 PROMPT_COMMAND
}

load_prompt