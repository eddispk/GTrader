from telethon import TelegramClient, events
from telethon.sessions import StringSession
from dotenv import load_dotenv
import os, sys
from telethon.utils import get_peer_id
from telethon.tl.types import Channel, Chat

# Load envs both from container and project root (safe)
load_dotenv()

class Telegram():
    def __init__(self):
        api_id_raw = os.getenv('API_ID', '0')
        self.api_id = int(api_id_raw)
        self.api_hash = os.getenv('API_HASH', '')
        # Support multiple channels:
        raw_list = os.getenv('SIGNAL_CHANNELS', '').strip()
        self.channels = [x.strip().lstrip('@') for x in raw_list.split(',') if x.strip()]

        self.bot_name = (os.getenv('BOT_NAME', '') or '').strip().lstrip('@')
        self.session_str = (os.getenv('TELETHON_STRING_SESSION', '') or '').strip()

        print("[PY] starting…")
        print("[PY] API_ID:", self.api_id)
        print("[PY] CHANNELS:", self.channels)
        print("[PY] BOT_NAME:", self.bot_name)

        if self.session_str:
            self.client = TelegramClient(StringSession(self.session_str), self.api_id, self.api_hash)
            print(f"[PY] RUN (string session)")
        else:
            self.client = TelegramClient("trading bot", self.api_id, self.api_hash)
            print(f"[PY] RUN (file session)")

    # --- replace your start() with this ---
    def start(self):
        if not self.channels:
            print("[PY][ERROR] no SIGNAL_CHANNEL(S) configured"); sys.exit(1)

        with self.client:
            self.client.loop.run_until_complete(self.client.connect())

            self.alias_by_id = {}   # peer id (-100...) -> label shown in stamp
            targets = []
            for ch in self.channels:
                try:
                    # allow @username / invite / or numeric -100... id
                    ident = int(ch) if ch.lstrip("-").isdigit() else ch
                    ent = self.client.loop.run_until_complete(self.client.get_entity(ident))
                    pid = get_peer_id(ent)                  # <- canonical peer id (-100...)
                    targets.append(pid)                     # listen by peer id

                    alias = getattr(ent, "username", None)
                    label = ("@" + alias) if alias else str(pid)  # use @ if public, else numeric id
                    self.alias_by_id[pid] = label
                    print(f"[PY] resolved: {ch} -> pid={pid} alias={label}")
                except Exception as e:
                    print(f"[PY][ERROR] cannot resolve '{ch}': {e}")
                    sys.exit(1)

            self.client.add_event_handler(self._forward, events.NewMessage(chats=targets))
            print(f"[PY] listening on {targets} -> @{self.bot_name}")
            self.client.run_until_disconnected()

    # --- keep _forward, but ensure this small fallback is present ---
    async def _forward(self, event):
        try:
            msg = event.message
            text = (msg.message or "").strip() or (event.raw_text or "").strip()
            if not text:
                print("[PY] empty; skip")
                return

            chat_id = event.chat_id  # peer id (-100...)
            label = self.alias_by_id.get(chat_id, str(chat_id))  # fallback: numeric id

            stamped = f"[CH:{label}] {text}"
            await self.client.send_message(self.bot_name, message=stamped)
            print(f"[PY] forwarded from {label} to @{self.bot_name}")
        except Exception as e:
            print(f"[PY][ERROR] forward failed: {e}")
            
if __name__ == '__main__':
    print("[PY] Run Python")
    if len(sys.argv) > 1 and sys.argv[1].lower() == "list":
    # Print all dialogs with numeric IDs so you can copy the channel id
        client = Telegram().client
        with client:
            client.loop.run_until_complete(client.connect())

            async def _list():
                async for d in client.iter_dialogs():
                    ent = d.entity
                    if isinstance(ent, (Channel, Chat)):
                        pid = get_peer_id(ent)  # e.g. -1001234567890 for channels
                        uname = getattr(ent, "username", None)
                        alias = ("@" + uname) if uname else ""
                        print(f"{pid}\t{alias}\t{d.name}")

            client.loop.run_until_complete(_list())
        sys.exit(0)
    Telegram().start()