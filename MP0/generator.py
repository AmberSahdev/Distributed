'''
Events can be generated using this Python script: generator.py.
For this assignments, events are represented as random strings of characters.
Each event is preceded by the generation timestamp, expressed as a fractional
number of seconds since 1970.

The argument to generator.py is the rate at which events occur specified in
hertz (i.e., events / second).
The events are exponentially distributed and so you could receive a burst
of events spaced closer together.
'''
from os import urandom
from hashlib import sha256
from random import expovariate
import time
import sys

if len(sys.argv) > 1:
    rate = float(sys.argv[1])
else:
    rate = 1.0 # default rate: 1 Hz

if len(sys.argv) > 2:
    max_events = int(sys.argv[2])
else:
    max_events = None

event_count = 0
while max_events is None or event_count < max_events:
    event_count += 1
    print("%s %s" % (time.time(), sha256(urandom(20)).hexdigest()))
    time.sleep(expovariate(rate))
