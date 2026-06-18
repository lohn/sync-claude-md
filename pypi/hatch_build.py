import os

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class CustomBuildHook(BuildHookInterface):
    def initialize(self, version, build_data):
        binary_path = os.environ["BINARY_PATH"]
        script_name = os.environ["SCRIPT_NAME"]
        wheel_tag = os.environ["WHEEL_TAG"]

        build_data["pure_python"] = False
        build_data["tag"] = wheel_tag
        build_data.setdefault("shared_scripts", {})[binary_path] = script_name
