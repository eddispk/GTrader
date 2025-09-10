from telethon import TelegramClient, events
from telethon.sessions import StringSession
from dotenv import load_dotenv
import os, sys

# Load envs both from container and project root (safe)
load_dotenv()

class Telegram():
    def __init__(self):
        api_id_raw = os.getenv('API_ID', '0')
        self.api_id = int(api_id_raw)
        self.api_hash = os.getenv('API_HASH', '')
        # Support multiple channels:
        # - SIGNAL_CHANNELS="chanA,chanB"
        # - or SIGNAL_CHANNEL + SIGNAL_CHANNEL_2
        raw_list = os.getenv('SIGNAL_CHANNELS', '').strip()
        c1 = (os.getenv('SIGNAL_CHANNEL', '') or '').strip()
        c2 = (os.getenv('SIGNAL_CHANNEL_2', '') or '').strip()
        if raw_list:
            self.channels = [x.strip().lstrip('@') for x in raw_list.split(',') if x.strip()]
        else:
            self.channels = [x.lstrip('@') for x in [c1, c2] if x]

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

    async def _forward(self, event):
        try:
            msg = event.message
            text = (msg.message or "").strip() or (event.raw_text or "").strip()
            print(f"[PY] NEW MSG chat_id={event.chat_id} kind={'media' if msg.media else 'text'} len={len(text)}")
            if not text:
                print("[PY] empty; skip")
                return
            await self.client.send_message(self.bot_name, message=text)
            print(f"[PY] forwarded to @{self.bot_name}")
        except Exception as e:
            print(f"[PY][ERROR] forward failed: {e}")

    def start(self):
        if not self.channels:
            print("[PY][ERROR] no SIGNAL_CHANNEL(S) configured"); sys.exit(1)

        with self.client:
            self.client.loop.run_until_complete(self.client.connect())

            # resolve all channels → ids
            targets = []
            for ch in self.channels:
                try:
                    ent = self.client.loop.run_until_complete(self.client.get_entity(ch))
                    targets.append(ent.id)
                    print(f"[PY] resolved: @{ch} -> id={ent.id}")
                except Exception as e:
                    print(f"[PY][ERROR] cannot resolve '{ch}': {e}")
                    sys.exit(1)

            # one handler that listens to ALL target ids
            self.client.add_event_handler(self._forward, events.NewMessage(chats=targets))
            print(f"[PY] listening on {targets} -> @{self.bot_name}")
            self.client.run_until_disconnected()

if __name__ == '__main__':
    print("[PY] Run Python")
    Telegram().start()