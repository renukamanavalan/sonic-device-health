import os
from setuptools import setup, find_packages

setup_requirements = ['pytest-runner']

test_requirements = ['pytest>=3']

# read me
with open('README.rst') as readme_file:
    readme = readme_file.read()

setup(
    author="LoM-dev",
    author_email='remanava@microsoft.com',
    python_requires='>=3.8',
    classifiers=[
        'Development Status :: 2 - Pre-Alpha',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: GNU General Public License v3 (GPLv3)',
        'Natural Language :: English',
        'Programming Language :: Python :: 3.8',
    ],
    description="Package contains LoM container modules",
    tests_require=[
        'pytest',
        'pytest-cov',
    ],
    install_requires=['netaddr', 'pyyaml'],
    license="GNU General Public License v3",
    long_description=readme + '\n\n',
    include_package_data=True,
    name='DH_LoM',
    py_modules=[],
    packages=find_packages(),
    setup_requires=setup_requirements,
    version='1.0.0',
    scripts=['plugin_manager/engine_if.py',
        'common/common.py',
        'common/gvars.py',
        'hal/DBAccess.py',
        'hal/sonic/ClientDBAccess.py',
        'hal/sonic/DBServer.py',
        'plugin_manager/engine_if.py',
        'plugin_manager/pluginMgr.py'],
    zip_safe=False,
)

#     @url='https://github.com/Azure/sonic-buildimage',
