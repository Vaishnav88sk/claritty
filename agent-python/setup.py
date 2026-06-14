from setuptools import setup, find_packages

def parse_requirements():
    try:
        with open("requirements.txt", "r") as f:
            return [line.strip() for line in f if line.strip() and not line.startswith("#")]
    except FileNotFoundError:
        return []

setup(
    name="claritty_sre",
    version="2.0.0",
    description="Claritty AI-SRE Engine CLI",
    author="Claritty",
    packages=find_packages(),
    include_package_data=True,
    install_requires=parse_requirements(),
    entry_points={
        "console_scripts": [
            "clarctl=claritty_sre.cli:cli",
        ],
    },
    python_requires=">=3.10",
)
