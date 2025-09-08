from telethon import TelegramClient, events
from dotenv import load_dotenv
import os
import sys


class Telegram():
    def __init__(self):
        if len(sys.argv) > 1:
            load_dotenv("../.env")
        self.api_id = os.getenv('API_ID')
        self.api_hash = os.getenv('API_HASH')
        self.id_channel = os.getenv('ID_CHANNEL')
        self.my_channel = os.getenv('SIGNAL_CHANNEL')
        self.bot_name = os.getenv('BOT_NAME')
        self.session = "trading bot"
        self.proxy = None
        self.msg = ""
        self.client = TelegramClient(self.session, self.api_id, self.api_hash, proxy=self.proxy)

    async def handler(self, update):
        t = update.raw_text
        print(t)
        if t != self.msg:
            await self.client.send_message(self.bot_name, message=t)
            self.msg = t
        return t

    def start(self):
            # Register the update handler so that it gets called
        with self.client:
            t = self.client.add_event_handler(self.handler, events.NewMessage(chats=self.my_channel))
            if len(sys.argv) > 1:
                exit(0)
            print(t)
        
            # Run the client until Ctrl+C is pressed, or the client disconnects
            self.client.run_until_disconnected()

if __name__ == '__main__':
    print("Run Python")
    telegram = Telegram()
    print("Class init....")
    telegram.start()
    
