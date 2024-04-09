import os, sys
# os.environ['HTTP_PROXY'] = '127.0.0.1:9999'
# os.environ['HTTPS_PROXY'] = '127.0.0.1:9999'

# request libarry uses certs from https://certifiio.readthedocs.io/en/latest/
# https://requests.readthedocs.io/en/latest/user/advanced/#ca-certificates
# proxies https://requests.readthedocs.io/en/latest/user/advanced/#proxies

# # Use a pipeline as a high-level helper
# from transformers import pipeline

# pipe = pipeline("image-to-text", model="microsoft/trocr-small-handwritten")

from requests.utils import DEFAULT_CA_BUNDLE_PATH
#os.environ['REQUESTS_CA_BUNDLE']="../certs/ca.cert"

print(DEFAULT_CA_BUNDLE_PATH)


# Downloaded under ~/.cache/huggingface/hub/

# Load model directly
from transformers import AutoTokenizer, AutoModel

# microsoft/conditional-detr-resnet-50
tokenizer = AutoTokenizer.from_pretrained("microsoft/trocr-small-handwritten")
model = AutoModel.from_pretrained("microsoft/trocr-small-handwritten")