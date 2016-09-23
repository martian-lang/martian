import martian
import json
import os
import random

__MRO__ = """
stage ADD_KEY(
    in string key,
    in string value,
    in json start,
    in string failfile,
    out json result,
    src py "stages/add_key",
)
"""

def main(args, outs):
  failfile = os.path.abspath(os.path.join(
      os.path.dirname(__file__), '..', '..', args.failfile))
  if args.failfile and os.path.isfile(failfile):
    s = ''
    with open(failfile, 'r') as f:
      s = f.read()
    os.unlink(failfile)
    os.kill(os.getpid(), int(s))
  if args.start:
    with open(args.start, 'r') as inpf:
      s = json.load(inpf)
  else:
    s = {}
  s[args.key]=args.value
  with open(outs.result, 'w') as outf:
    json.dump(s, outf, indent=2)
