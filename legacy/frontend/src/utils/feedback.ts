import { message as staticMessage, Modal as staticModal } from "antd";
import type { MessageInstance } from "antd/es/message/interface";

type ModalConfirmApi = Pick<typeof staticModal, "confirm">;

let messageApi: MessageInstance = staticMessage;
let modalApi: ModalConfirmApi = staticModal;

export function setMessageApi(api: MessageInstance): void {
  messageApi = api;
}

export function getMessageApi(): MessageInstance {
  return messageApi;
}

export function setModalApi(api: ModalConfirmApi): void {
  modalApi = api;
}

export function getModalApi(): ModalConfirmApi {
  return modalApi;
}
