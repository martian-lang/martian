#!/usr/bin/env python
#
# Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
#
# check code for an individual stage
#
import os
import sys
import json
import traceback

# Parse json from STDIN.
input = json.loads(sys.stdin.read())

# Import the stage code.
code_path = input['codePath']
sys.path.append(os.path.dirname(code_path))

try:
    stage_code = __import__(os.path.basename(code_path))
except Exception as e:
    sys.stdout.write(json.dumps({ 'error': traceback.format_exc() }))
    exit(1)

# Push output to STDOUT.
in_params = []
out_params = []
try:
    in_params.extend(stage_code.in_params)
except:
    pass
try:
    out_params.extend(stage_code.out_params)
except:
    pass
sys.stdout.write(json.dumps({ 
    'exports': dir(stage_code),
    'in_params': in_params,
    'out_params': out_params
}))