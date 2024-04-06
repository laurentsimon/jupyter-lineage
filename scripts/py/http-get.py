
# import os
# os.environ['HTTP_PROXY'] = 'localhost:9999'
# os.environ['HTTPS_PROXY'] = 'localhost:9999'

#from urllib.request import urlopen

# import requests
# import urllib
# response = requests.get("http://www.google.com")
# print (response.Body)

# import urllib3

# # Creating a PoolManager instance for sending requests.
# http = urllib3.PoolManager()

# # Sending a GET request and getting back response as HTTPResponse object.
# resp = http.request("GET", "http://www.google.com")

# # Print the returned data.
# print(resp.data)

import os, sys
os.environ['HTTP_PROXY'] = '127.0.0.1:9999'
os.environ['HTTPS_PROXY'] = '127.0.0.1:9999'

import urllib.request
import requests

if len(sys.argv[1:]) != 1:
   raise ValueError(f"expected 1 argument, got {len(sys.argv[1:])}")

url = sys.argv[1]


# Uses urllib
# print(urllib.request.getproxies())
# with urllib.request.urlopen(url) as response:
#    html = response.read()
#    print(html)

# Uses requests.
from requests.utils import DEFAULT_CA_BUNDLE_PATH
import certifi
print(certifi.where())
#os.environ['REQUESTS_CA_BUNDLE']="/etc/ssl/certs/jupyter-proxy.pem"
os.environ['REQUESTS_CA_BUNDLE']="../certs/ca.cert"
print(DEFAULT_CA_BUNDLE_PATH)
import certifi
print(certifi.where())
#r = requests.get(url, verify="../certs/ca.cert")
r = requests.get(url)
print(r)
print(r.content)