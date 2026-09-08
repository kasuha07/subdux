import unittest

import subdux_settings as settings


class RevisionPlanTests(unittest.TestCase):
    def test_update_then_reorder_uses_next_revision(self):
        current = {
            "currencies": [{"id": 7, "revision": 4, "code": "USD", "symbol": "$", "alias": "old", "sort_order": 3}],
        }
        plan = settings.build_plan(current, {"currencies": [{"code": "USD", "symbol": "$", "alias": "new"}]})
        self.assertEqual(plan["version"], 2)
        update, reorder = plan["actions"]
        self.assertEqual(update["payload"]["revision"], 4)
        self.assertEqual(reorder["payload"], [{"id": 7, "sort_order": 0, "revision": 5}])
        self.assertEqual(current["currencies"][0]["revision"], 4)

    def test_cleanup_binds_delete_to_reviewed_revision(self):
        current = {"subscriptions": [], "categories": [{"id": 3, "revision": 9, "name": "unused"}]}
        plan = settings.build_cleanup_plan(current)
        action = plan["actions"][0]
        self.assertEqual(action["revision"], 9)

        class Client:
            def request(self, method, path, payload=None):
                return method, path, payload

        self.assertEqual(settings.apply_action(Client(), action), ("DELETE", "/api/categories/3?revision=9", None))

    def test_missing_revision_cannot_produce_an_applicable_plan(self):
        current = {"subscriptions": [], "categories": [{"id": 3, "name": "unused"}]}
        with self.assertRaises(settings.SubduxError):
            settings.build_cleanup_plan(current)


if __name__ == "__main__":
    unittest.main()
