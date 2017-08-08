from setuptools import setup, find_packages
# To use a consistent encoding
from codecs import open
from os import path

here = path.abspath(path.dirname(__file__))

# Get the long description from the README file
with open(path.join(here, 'README.md'), encoding='utf-8') as f:
    long_description = f.read()

setup (
    name='0-core-client',
    version='1.1.0-alpha-7',
    description='Zero-OS 0-core client',
    long_description=long_description,
    url='https://github.com/zero-os/0-core',
    author='Muhamad Azmy',
    author_email='muhamada@greenitglobe.com',
    license='Apache 2.0',
    namespaces=['zeroos'],
    packages=find_packages(),
    install_requires=['redis>=2.10.5'],
)
