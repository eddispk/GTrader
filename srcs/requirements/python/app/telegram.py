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
        self.sig_channel = (os.getenv('SIGNAL_CHANNEL', '') or '').strip()
        self.bot_name = (os.getenv('BOT_NAME', '') or '').strip()
        self.session_str = (os.getenv('TELETHON_STRING_SESSION', '') or '').strip()

        print("[PY] starting…")
        print("[PY] API_ID:", self.api_id)
        print("[PY] SIGNAL_CHANNEL:", self.sig_channel)
        print("[PY] BOT_NAME:", self.bot_name)

        if self.sig_channel.startswith('@'): self.sig_channel = self.sig_channel[1:]
        if self.bot_name.startswith('@'): self.bot_name = self.bot_name[1:]

        if self.session_str:
            self.client = TelegramClient(StringSession(self.session_str), self.api_id, self.api_hash)
            print(f"[PY] RUN (string session). sig_channel={self.sig_channel} -> bot={self.bot_name}")
        else:
            # fallback to file session "trading bot"
            self.client = TelegramClient("trading bot", self.api_id, self.api_hash)
            print(f"[PY] RUN (file session). sig_channel={self.sig_channel} -> bot={self.bot_name}")

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
        with self.client:
            self.client.loop.run_until_complete(self.client.connect())
            try:
                entity = self.client.loop.run_until_complete(self.client.get_entity(self.sig_channel))
                target = entity.id
                print(f"[PY] resolved SIGNAL_CHANNEL: id={entity.id} username={getattr(entity,'username',None)} title={getattr(entity,'title',None)}")
            except Exception as e:
                print(f"[PY][ERROR] cannot resolve SIGNAL_CHANNEL '{self.sig_channel}': {e}")
                sys.exit(1)

            self.client.add_event_handler(self._forward, events.NewMessage(chats=target))
            print(f"[PY] listening on id={target} (@{self.sig_channel}) -> @{self.bot_name}")
            self.client.run_until_disconnected()

if __name__ == '__main__':
    print("[PY] Run Python")
    Telegram().start()