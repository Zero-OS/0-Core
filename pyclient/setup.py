from setuptools import setup, find_packages
# To use a consistent encoding
from codecs import open
from os import path

here = path.abspath(path.dirname(__file__))

# Get the long description from the README file
with open(path.join(here, 'README'), encoding='utf-8') as f:
    long_description = f.read()

setup (
    name='g8core',
    version='0.11.1',
    description='G8OS cores client',
    long_description=long_description,
    url='https://github.com/g8os/core0',
    author='Muhamad Azmy',
    author_email='muhamad.azmy@greenitglobe.com',
    license='Apache 2.0',
    packages=find_packages(),
    install_requires=['redis>=2.10.5'],
)
