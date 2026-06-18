import { message as staticMessage } from "antd";
import type { MessageInstance } from "antd/es/message/interface";

let messageApi: MessageInstance = staticMessage;

export function setMessageApi(api: MessageInstance): void {
  messageApi = api;
}

export function getMessageApi(): MessageInstance {
  return messageApi;
}
