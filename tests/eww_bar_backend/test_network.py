import sys
import unittest
import unittest.mock
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
SCRIPTS_DIR = REPO_ROOT / "modules" / "desktop" / "home" / "hyprland" / "eww" / "scripts"
sys.path.insert(0, str(SCRIPTS_DIR))

from eww_bar_backend import collectors, control  # noqa: E402


class NetworkRadioTests(unittest.TestCase):
    def test_enabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="enabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "true")

    def test_disabled(self):
        with unittest.mock.patch.object(collectors, "run_text", return_value="disabled\n"):
            self.assertEqual(collectors.network_radio_enabled(), "false")

    def test_state_includes_wifi_enabled(self):
        with unittest.mock.patch.object(
            collectors, "network_connection_state", return_value={"text": "x", "class": "wifi"}
        ):
            with unittest.mock.patch.object(collectors, "network_radio_enabled", return_value="true"):
                state = collectors.network_state()
        self.assertEqual(state["wifi_enabled"], "true")
        self.assertEqual(state["class"], "wifi")


class WifiToggleTests(unittest.TestCase):
    def _capture(self):
        calls = []

        def fake_run(command, **_kwargs):
            calls.append(command)

            class Result:
                returncode = 0

            return Result()

        return calls, fake_run

    def test_toggle_off_when_on(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="true"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "off"]])

    def test_toggle_on_when_off(self):
        calls, fake_run = self._capture()
        with unittest.mock.patch.object(control.subprocess, "run", side_effect=fake_run):
            with unittest.mock.patch.object(control, "network_radio_enabled", return_value="false"):
                with unittest.mock.patch.object(control, "network_state", return_value={}):
                    control.toggle_wifi()
        self.assertEqual(calls, [["nmcli", "radio", "wifi", "on"]])

    def test_arg_parsing(self):
        self.assertEqual(
            control.control_payload_from_args(["network", "wifi-toggle"]),
            {"command": "network", "action": "wifi-toggle"},
        )


ROUTE_TEXT = (
    "default via 192.168.0.1 dev eno1 proto dhcp src 192.168.0.88 metric 100\n"
    "192.168.0.0/24 dev eno1 proto kernel scope link src 192.168.0.88 metric 100\n"
)

ADDR_TEXT = (
    "1: lo    inet 127.0.0.1/8 scope host lo\n"
    "2: eno1    inet 192.168.0.88/24 brd 192.168.0.255 scope global dynamic eno1\n"
)


class LinkFallbackTests(unittest.TestCase):
    def test_finds_default_route_device(self):
        self.assertEqual(collectors.default_route_device(ROUTE_TEXT), "eno1")

    def test_no_default_route(self):
        self.assertEqual(
            collectors.default_route_device("192.168.0.0/24 dev eno1 scope link\n"), ""
        )

    def test_device_ipv4(self):
        self.assertEqual(collectors.device_ipv4(ADDR_TEXT, "eno1"), "192.168.0.88")

    def test_device_ipv4_missing(self):
        self.assertEqual(collectors.device_ipv4(ADDR_TEXT, "wlan0"), "")

    def test_link_state_reports_route_owner(self):
        state = collectors.link_state_from_text(ROUTE_TEXT, ADDR_TEXT)
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])
        self.assertIn("192.168.0.88", state["tooltip"])
        self.assertIn("NetworkManager", state["tooltip"])

    def test_link_state_none_without_default_route(self):
        self.assertIsNone(collectors.link_state_from_text("", ADDR_TEXT))


class NetworkManagerDownTests(unittest.TestCase):
    def _fake_run_text(self, responses):
        def fake(command, **_kwargs):
            return responses.get(tuple(command), "")

        return fake

    def test_falls_back_to_link_state_when_nmcli_is_dead(self):
        responses = {
            ("ip", "route"): ROUTE_TEXT,
            ("ip", "-o", "-4", "addr", "show"): ADDR_TEXT,
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "degraded")
        self.assertIn("eno1", state["text"])

    def test_disconnected_when_nmcli_dead_and_no_route(self):
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text({})
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "disconnected")

    def test_ethernet_wins_when_it_owns_the_default_route(self):
        responses = {
            ("nmcli", "-t", "-f", "DEVICE,TYPE,STATE", "dev", "status"): (
                "wlan0:wifi:connected\neno1:ethernet:connected\n"
            ),
            ("ip", "route"): ROUTE_TEXT,
            ("nmcli", "-t", "-f", "IP4.ADDRESS", "dev", "show", "eno1"): (
                "IP4.ADDRESS[1]:192.168.0.88/24\n"
            ),
        }
        with unittest.mock.patch.object(
            collectors, "run_text", side_effect=self._fake_run_text(responses)
        ):
            state = collectors.network_connection_state()
        self.assertEqual(state["class"], "ethernet")
        self.assertIn("eno1", state["text"])


if __name__ == "__main__":
    unittest.main()
