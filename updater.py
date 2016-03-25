#!/usr/bin/env python

import os
import time
import sys
import platform
import shutil
import subprocess

if sys.version_info.major == 3:
    import urllib.request as urllib
else:
    import urllib2 as urllib

strfilepath = os.path.realpath(__file__)
WORKPATH = "%s/" % (os.path.dirname(strfilepath),)
os.chdir(WORKPATH)
print(WORKPATH)


def DownloadNewVersion(url_str):
    print('Download: %s' % url_str)
    f = urllib.urlopen(url_str) 
    with open(os.path.basename(url_str), "wb") as code:
       code.write(f.read())

def KillRunning():
    if platform.system() == 'Windows':
        os.system('taskkill /F /IM DCS.exe')
    else:
        os.system('pkill DCS')
    
def Replace(new_name):
    if platform.system() == 'Windows':
        shutil.copy('DCS.exe', 'DCS.exe.bak')
        os.remove('DCS.exe')
        os.rename(new_name, 'DCS.exe')

def Run():
    if platform.system() == 'Windows':
        exe_cmd = 'DCS.exe > DCSLog.log'
    else:
        exe_cmd = 'DCS'
    try:
        log_fp == open('DCSLog.log', 'w')
        p = subprocess.Popen(exe_cmd, stdout=log_fp, stderr=log_fp)
        print(p.pid)
        os.remove(exe_cmd+'.bak')
    except:
        print('Update failed.')
        print('Rerun old version.')
        os.remove(exe_cmd)
        os.rename(exe_cmd+'.bak', exe_cmd)
        p = subprocess.Popen(exe_cmd)


def main(url_str):
    DownloadNewVersion(url_str)

    KillRunning()

    Replace(os.path.basename(url_str))

    Run()

if __name__ == '__main__':
    url_str = sys.argv[1]
    main(url_str)

#

