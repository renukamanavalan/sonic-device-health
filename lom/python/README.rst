LoM Python code


Simple test:
Python 3.7.3 (default, Jan 22 2021, 20:04:44) 
[GCC 8.3.0] on linux
Type "help", "copyright", "credits" or "license" for more information.
>>> from ctypes import *
>>> lomLib = CDLL("/home/admin/bin/cmn_c_lib.so")
>>> fnrcl = lomLib.RegisterClientC
>>> cl_name = "test".encode("utf-8")
>>> fnrcl(cl_name)
lomipc/client_transport.go:88:Registered client (test)
29397968


lom/python/src/common/engine_apis.py
-----------------------------------

1. Loads C-lib
2. For all C-lib APIs
   Load each API object against uniq int based index per API
   Ech entry explicitly describes the args and result

3. call_lom_lib with index & args 
   Calls the underlying API via cached data
