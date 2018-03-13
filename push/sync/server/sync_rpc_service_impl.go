/*
 *  Copyright (c) 2017, https://github.com/nebulaim
 *  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package server

import (
	"github.com/golang/glog"
	"github.com/nebulaim/telegramd/baselib/base"
	"github.com/nebulaim/telegramd/baselib/logger"
	base2 "github.com/nebulaim/telegramd/biz_model/base"
	"github.com/nebulaim/telegramd/biz_model/model"
	"github.com/nebulaim/telegramd/mtproto"
	"golang.org/x/net/context"
	"time"
	"fmt"
	"sync"
)

type SyncServiceImpl struct {
	// status *model.OnlineStatusModel

	mu sync.RWMutex
	s  *syncServer
	// TODO(@benqi): 多个连接
	// updates map[int32]chan *zproto.PushUpdatesNotify
}

func NewSyncService(sync2 *syncServer) *SyncServiceImpl {
	s := &SyncServiceImpl{s: sync2}
	// s.status = status
	// s.updates = make(map[int32]chan *zproto.PushUpdatesNotify)
	return s
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// 推送给该用户所有设备
func (s *SyncServiceImpl) pushToUserUpdates(userId int32, updates *mtproto.Updates) {
	// pushRawData := updates.Encode()

	statusList, _ := model.GetOnlineStatusModel().GetOnlineByUserId(userId)
	ss := make(map[int32][]*model.SessionStatus)
	for _, status := range statusList {
		if _, ok := ss[status.ServerId]; !ok {
			ss[status.ServerId] = []*model.SessionStatus{}
		}
		// 会不会出问题？？
		ss[status.ServerId] = append(ss[status.ServerId], status)
	}

	rawData := updates.Encode()
	for k, ss3 := range ss {
		//// glog.Infof("DeliveryUpdates: k: {%v}, v: {%v}", k, ss3)
		//go s.withReadLock(func() {
		// _ = k
		for _, ss4 := range ss3 {
			pushData := &mtproto.PushUpdatesData{
				ClientId: &mtproto.PushClientID{
					AuthKeyId:       ss4.AuthKeyId,
					SessionId:       ss4.SessionId,
					NetlibSessionId: ss4.NetlibSessionId,
				},
				// RawDataHeader:
				RawData: rawData,
			}
			// _ = pushData
			s.s.sendToSessionServer(int(k), pushData)
		}
		//})
	}
}

// 推送给该用户剔除指定设备后的所有设备
func (s *SyncServiceImpl) pushToUserUpdatesNotMe(userId int32, sessionId int64, updates *mtproto.Updates) {
	// pushRawData := updates.Encode()

	statusList, _ := model.GetOnlineStatusModel().GetOnlineByUserId(userId)
	ss := make(map[int32][]*model.SessionStatus)
	for _, status := range statusList {
		if _, ok := ss[status.ServerId]; !ok {
			ss[status.ServerId] = []*model.SessionStatus{}
		}
		// 会不会出问题？？
		ss[status.ServerId] = append(ss[status.ServerId], status)
	}

	rawData := updates.Encode()
	for k, ss3 := range ss {
		_ = k
		for _, ss4 := range ss3 {
			if ss4.SessionId != sessionId {
				pushData := &mtproto.PushUpdatesData{
					ClientId: &mtproto.PushClientID{
						AuthKeyId:       ss4.AuthKeyId,
						SessionId:       ss4.SessionId,
						NetlibSessionId: ss4.NetlibSessionId,
					},
					// RawDataHeader:
					RawData: rawData,
				}
				_ = pushData
				// s.s.sendToSessionServer(int(k), pushData)
			}
		}
	}
}

//SyncUpdateShortMessage(context.Context, *SyncShortMessageRequest) (*ClientUpdatesState, error)
////////////////////////////////////////////////////////////////////////////////////////////////////
func (s *SyncServiceImpl) SyncUpdateShortMessage(ctx context.Context, request *mtproto.SyncShortMessageRequest) (reply *mtproto.ClientUpdatesState, err error) {
	glog.Infof("syncUpdateShortMessage - request: {%v}", request)

	var (
		userId, peerId int32
		pts, ptsCount int32
		updateType int32
	)

	userId = request.GetPushtoUserId()
	shortMessage := request.GetPushData()
	peerId = request.GetPeerId()
	updateType = model.PTS_MESSAGE_OUTBOX

	pts = int32(model.GetSequenceModel().NextPtsId(base.Int32ToString(userId)))
	ptsCount = int32(1)
	shortMessage.SetPts(pts)
	shortMessage.SetPtsCount(ptsCount)

	// save pts
	model.GetUpdatesModel().AddPtsToUpdatesQueue(userId, pts, base2.PEER_USER, peerId, updateType, shortMessage.GetId(), 0)

	// push
	// if request.GetPushType() == mtproto.SyncType_SYNC_TYPE_USER_NOTME {
	s.pushToUserUpdatesNotMe(userId, request.GetClientId().GetSessionId(), shortMessage.To_Updates())
	// } else {
	//	s.pushToUserUpdates(userId, shortMessage.To_Updates())
	//}

	reply = &mtproto.ClientUpdatesState{
		Pts:      pts,
		PtsCount: ptsCount,
		Date:     int32(time.Now().Unix()),
	}

	glog.Infof("DeliveryPushUpdateShortMessage - reply: %s", logger.JsonDebugData(reply))
	return
}

//PushUpdateShortMessage(context.Context, *UpdateShortMessageRequest) (*VoidRsp, error)
func (s *SyncServiceImpl) PushUpdateShortMessage(ctx context.Context, request *mtproto.UpdateShortMessageRequest) (reply *mtproto.VoidRsp, err error) {
	glog.Infof("pushUpdateShortMessage - request: {%v}", request)

	var (
		userId, peerId int32
		pts, ptsCount int32
		updateType int32
	)

	userId = request.GetPushtoUserId()
	shortMessage := request.GetPushData()
	peerId = request.GetPeerId()
	updateType = model.PTS_MESSAGE_INBOX

	pts = int32(model.GetSequenceModel().NextPtsId(base.Int32ToString(userId)))
	ptsCount = int32(1)
	shortMessage.SetPts(pts)
	shortMessage.SetPtsCount(ptsCount)

	// save pts
	model.GetUpdatesModel().AddPtsToUpdatesQueue(userId, pts, base2.PEER_USER, peerId, updateType, shortMessage.GetId(), 0)
	// push
	s.pushToUserUpdates(userId, shortMessage.To_Updates())

	reply = &mtproto.VoidRsp{}
	glog.Infof("pushUpdateShortMessage - reply: %s", logger.JsonDebugData(reply))
	return
}

//SyncUpdateShortChatMessage(context.Context, *SyncShortChatMessageRequest) (*ClientUpdatesState, error)
func (s *SyncServiceImpl) SyncUpdateShortChatMessage(ctx context.Context, request *mtproto.SyncShortChatMessageRequest) (reply *mtproto.ClientUpdatesState, err error) {
	glog.Infof("syncUpdateShortChatMessage - request: {%v}", request)

	var (
		pts, ptsCount int32
		updateType int32
	)

	shortChatMessage := request.GetPushData()
	// var outgoing = request.GetSenderUserId() == pushData.GetPushUserId()
	updateType = model.PTS_MESSAGE_OUTBOX

	pts = int32(model.GetSequenceModel().NextPtsId(base.Int32ToString(request.GetPushtoUserId())))
	ptsCount = int32(1)
	shortChatMessage.SetPts(pts)
	shortChatMessage.SetPtsCount(ptsCount)

	// save pts
	model.GetUpdatesModel().AddPtsToUpdatesQueue(request.GetPushtoUserId(), pts, base2.PEER_CHAT, request.GetPeerChatId(), updateType, shortChatMessage.GetId(), 0)
	s.pushToUserUpdatesNotMe(request.GetPushtoUserId(), request.GetClientId().GetSessionId(), shortChatMessage.To_Updates())

	reply = &mtproto.ClientUpdatesState{
		Pts:      pts,
		PtsCount: ptsCount,
		Date:     int32(time.Now().Unix()),
	}
	glog.Infof("syncUpdateShortChatMessage - reply: %s", logger.JsonDebugData(reply))
	return
}

//PushUpdateShortChatMessage(context.Context, *UpdateShortChatMessageRequest) (*VoidRsp, error)
func (s *SyncServiceImpl) PushUpdateShortChatMessage(ctx context.Context, request *mtproto.UpdateShortChatMessageRequest) (reply *mtproto.VoidRsp, err error) {
	glog.Infof("pushUpdateShortChatMessage - request: {%v}", request)

	var (
		pts, ptsCount int32
		updateType int32
	)

	shortChatMessage := request.GetPushData()
	updateType = model.PTS_MESSAGE_INBOX

	pts = int32(model.GetSequenceModel().NextPtsId(base.Int32ToString(request.GetPushtoUserId())))
	ptsCount = int32(1)
	shortChatMessage.SetPts(pts)
	shortChatMessage.SetPtsCount(ptsCount)

	// save pts
	model.GetUpdatesModel().AddPtsToUpdatesQueue(request.GetPushtoUserId(), pts, base2.PEER_CHAT, request.GetPeerChatId(), updateType, shortChatMessage.GetId(), 0)
	s.pushToUserUpdates(request.GetPushtoUserId(), shortChatMessage.To_Updates())

	reply = &mtproto.VoidRsp{}
	glog.Infof("pushUpdateShortChatMessage - reply: %s", logger.JsonDebugData(reply))
	return
}

//SyncUpdateMessageData(context.Context, *SyncUpdateMessageRequest) (*ClientUpdatesState, error)
func (s *SyncServiceImpl) SyncUpdateMessageData(ctx context.Context, request *mtproto.SyncUpdateMessageRequest) (reply *mtproto.ClientUpdatesState, err error) {
	glog.Infof("syncUpdateData - request: {%v}", request)

	// TODO(@benqi): Check deliver valid!
	update := request.GetPushData()

	var (
		pts, ptsCount int32
		updateType int32
	)

	switch update.GetConstructor() {
	case mtproto.TLConstructor_CRC32_updateReadHistoryInbox:
		updateType = model.PTS_READ_HISTORY_INBOX
		updateReadHistoryInbox := update.To_UpdateReadHistoryInbox()

		pts = int32(model.GetSequenceModel().NextPtsId(base.Int32ToString(request.GetPushtoUserId())))
		ptsCount = int32(1)

		updateReadHistoryInbox.SetPts(pts)
		updateReadHistoryInbox.SetPts(ptsCount)

		model.GetUpdatesModel().AddPtsToUpdatesQueue(request.GetPushtoUserId(), pts, base2.PEER_USER, request.GetPeerId(), updateType, 0, updateReadHistoryInbox.GetMaxId())

		updates := mtproto.NewTLUpdates()
		updates.SetSeq(0)
		updates.SetDate(int32(time.Now().Unix()))
		updates.SetUpdates([]*mtproto.Update{updateReadHistoryInbox.To_Update()})
		s.pushToUserUpdatesNotMe(request.GetPushtoUserId(), request.GetClientId().GetSessionId(), updates.To_Updates())

		reply = &mtproto.ClientUpdatesState{
			Pts:      pts,
			PtsCount: ptsCount,
			Date:     int32(time.Now().Unix()),
		}
		glog.Infof("sync updateReadHistoryInbox - reply: %s", logger.JsonDebugData(reply))

	default:
		err = fmt.Errorf("invalid update")
	}
	return
}

//PushUpdateMessageData(context.Context, *PushUpdateMessageRequest) (*VoidRsp, error)
func (s *SyncServiceImpl) PushUpdateMessageData(ctx context.Context, request *mtproto.PushUpdateMessageRequest) (reply *mtproto.VoidRsp, err error) {
	glog.Infof("pushUpdateData - request: {%v}", request)

	// TODO(@benqi): Check deliver valid!
	update := request.GetPushData()

	var (
		pts, ptsCount int32
		updateType int32
	)

	switch update.GetConstructor() {
	case mtproto.TLConstructor_CRC32_updateReadHistoryOutbox:
		updateType = model.PTS_READ_HISTORY_OUTBOX
		updateReadHistoryOutbox := update.To_UpdateReadHistoryOutbox()

		pts = int32(model.GetSequenceModel().NextPtsId(base.Int32ToString(request.GetPushtoUserId())))
		ptsCount = int32(1)
		updateReadHistoryOutbox.SetPts(pts)
		updateReadHistoryOutbox.SetPts(ptsCount)

		model.GetUpdatesModel().AddPtsToUpdatesQueue(request.GetPushtoUserId(), pts, base2.PEER_USER, request.GetPeerId(), updateType, 0, updateReadHistoryOutbox.GetMaxId())

		updates := mtproto.NewTLUpdates()
		updates.SetSeq(0)
		updates.SetDate(int32(time.Now().Unix()))
		updates.SetUpdates([]*mtproto.Update{updateReadHistoryOutbox.To_Update()})
		s.pushToUserUpdates(request.GetPushtoUserId(), updates.To_Updates())

		reply = &mtproto.VoidRsp{}
		glog.Infof("push updateReadHistoryOutbox - reply: %s", logger.JsonDebugData(reply))
	default:
		err = fmt.Errorf("invalid update")
	}

	return
}

//PushUpdatesData(context.Context, *PushUpdatesRequest) (*VoidRsp, error)
func (s *SyncServiceImpl) PushUpdatesData(ctx context.Context, request *mtproto.PushUpdatesRequest) (reply *mtproto.VoidRsp, err error) {
	glog.Infof("pushUpdateData - request: {%v}", request)

	// TODO(@benqi): Check deliver valid!
	// update := request.GetPushData()
	switch request.GetPushType() {
	case mtproto.SyncType_SYNC_TYPE_USER:
		s.pushToUserUpdates(request.GetPushtoUserId(), request.GetPushData().To_Updates())
	case mtproto.SyncType_SYNC_TYPE_USER_NOTME:
		// s.pushToUserUpdatesNotMe(request.GetPushtoUserId(), request.GetPushData().To_Updates())
	case mtproto.SyncType_SYNC_TYPE_AUTH_KEY:
	case mtproto.SyncType_SYNC_TYPE_AUTH_KEY_USER:
	case mtproto.SyncType_SYNC_TYPE_AUTH_KEY_USERNOTME:
	default:
	}

	reply = &mtproto.VoidRsp{}
	glog.Infof("push updateReadHistoryOutbox - reply: %s", logger.JsonDebugData(reply))
	return
}

//PushUpdateShortData(context.Context, *PushUpdateShortRequest) (*VoidRsp, error)
func (s *SyncServiceImpl) PushUpdateShortData(ctx context.Context, request *mtproto.PushUpdateShortRequest) (reply *mtproto.VoidRsp, err error) {
	glog.Infof("pushUpdateData - request: {%v}", request)

	// TODO(@benqi): Check deliver valid!
	// update := request.GetPushData()
	switch request.GetPushType() {
	case mtproto.SyncType_SYNC_TYPE_USER:
		s.pushToUserUpdates(request.GetPushtoUserId(), request.GetPushData().To_Updates())
	case mtproto.SyncType_SYNC_TYPE_USER_NOTME:
		// s.pushToUserUpdatesNotMe(request.GetPushtoUserId(), request.GetPushData().To_Updates())
	case mtproto.SyncType_SYNC_TYPE_AUTH_KEY:
	case mtproto.SyncType_SYNC_TYPE_AUTH_KEY_USER:
	case mtproto.SyncType_SYNC_TYPE_AUTH_KEY_USERNOTME:
	default:
	}

	reply = &mtproto.VoidRsp{}
	glog.Infof("push updateReadHistoryOutbox - reply: %s", logger.JsonDebugData(reply))
	return
}

/*
func (s *SyncServiceImpl) GetUpdatesData(ctx context.Context, request *zproto.GetUpdatesDataRequest) (reply *zproto.UpdatesDatasRsp, err error) {
	glog.Infof("GetUpdatesData - request: {%v}", request)

/ *
	// TODO(@benqi):
	md := grpc_util.RpcMetadataFromIncoming(ctx)
	glog.Infof("UpdatesGetDifference - metadata: %s, request: %s", logger.JsonDebugData(md), logger.JsonDebugData(request))
	difference := mtproto.NewTLUpdatesDifference()
	otherUpdates := []*mtproto.Update{}

	lastPts := request.GetPts()
	doList := dao.GetUserPtsUpdatesDAO(dao.DB_SLAVE).SelectByGtPts(md.UserId, request.GetPts())
	boxIDList := make([]int32, 0, len(doList))
	for _, do := range doList {
		switch do.UpdateType {
		case model.PTS_READ_HISTORY_OUTBOX:
			readHistoryOutbox := mtproto.NewTLUpdateReadHistoryOutbox()
			readHistoryOutbox.SetPeer(base.ToPeerByTypeAndID(do.PeerType, do.PeerId))
			readHistoryOutbox.SetMaxId(do.MaxMessageBoxId)
			readHistoryOutbox.SetPts(do.Pts)
			readHistoryOutbox.SetPtsCount(0)
			otherUpdates = append(otherUpdates, readHistoryOutbox.To_Update())
		case model.PTS_READ_HISTORY_INBOX:
			readHistoryInbox := mtproto.NewTLUpdateReadHistoryInbox()
			readHistoryInbox.SetPeer(base.ToPeerByTypeAndID(do.PeerType, do.PeerId))
			readHistoryInbox.SetMaxId(do.MaxMessageBoxId)
			readHistoryInbox.SetPts(do.Pts)
			readHistoryInbox.SetPtsCount(0)
			otherUpdates = append(otherUpdates, readHistoryInbox.To_Update())
		case model.PTS_MESSAGE_OUTBOX, model.PTS_MESSAGE_INBOX:
			boxIDList = append(boxIDList, do.MessageBoxId)
		}

		if do.Pts > lastPts {
			lastPts = do.Pts
		}
	}

	if len(boxIDList) > 0 {
		messages := model.GetMessageModel().GetMessagesByPeerAndMessageIdList2(md.UserId, boxIDList)
		// messages := model.GetMessageModel().GetMessagesByGtPts(md.UserId, request.Pts)
		userIdList := []int32{}
		chatIdList := []int32{}

		for _, m := range messages {
			switch m.GetConstructor()  {

			case mtproto.TLConstructor_CRC32_message:
				m2 := m.To_Message()
				userIdList = append(userIdList, m2.GetFromId())
				p := base.FromPeer(m2.GetToId())
				switch p.PeerType {
				case base.PEER_SELF, base.PEER_USER:
					userIdList = append(userIdList, p.PeerId)
				case base.PEER_CHAT:
					chatIdList = append(chatIdList, p.PeerId)
				case base.PEER_CHANNEL:
					// TODO(@benqi): add channel
				}
				//peer := base.FromPeer(m2.GetToId())
				//switch peer.PeerType {
				//case base.PEER_USER:
				//    userIdList = append(userIdList, peer.PeerId)
				//case base.PEER_CHAT:
				//    chatIdList = append(chatIdList, peer.PeerId)
				//case base.PEER_CHANNEL:
				//    // TODO(@benqi): add channel
				//}
			case mtproto.TLConstructor_CRC32_messageService:
				m2 := m.To_MessageService()
				userIdList = append(userIdList, m2.GetFromId())
				chatIdList = append(chatIdList, m2.GetToId().GetData2().GetChatId())
			case mtproto.TLConstructor_CRC32_messageEmpty:
			}
			difference.Data2.NewMessages = append(difference.Data2.NewMessages, m)
		}

		if len(userIdList) > 0 {
			usersList := model.GetUserModel().GetUserList(userIdList)
			for _, u := range usersList {
				if u.GetId() == md.UserId {
					u.SetSelf(true)
				} else {
					u.SetSelf(false)
				}
				u.SetContact(true)
				u.SetMutualContact(true)
				difference.Data2.Users = append(difference.Data2.Users, u.To_User())
			}
		}
	}

	difference.SetOtherUpdates(otherUpdates)

	state := mtproto.NewTLUpdatesState()
	// TODO(@benqi): Pts通过规则计算出来
	state.SetPts(lastPts)
	state.SetDate(int32(time.Now().Unix()))
	state.SetUnreadCount(0)
	state.SetSeq(int32(model.GetSequenceModel().CurrentSeqId(base2.Int32ToString(md.UserId))))

	difference.SetState(state.To_Updates_State())

	// dao.GetAuthUpdatesStateDAO(dao.DB_MASTER).UpdateState(request.GetPts(), request.GetQts(), request.GetDate(), md.AuthId, md.UserId)
	//glog.Infof("UpdatesGetDifference - reply: %s", difference)
	//return difference.To_Updates_Difference(), nil

* /
	glog.Infof("GetUpdatesData - reply {%v}", reply)
	return
}

//
//func (s *SyncServiceImpl) DeliveryUpdates2(ctx context.Context, request *mtproto.UpdatesRequest) (reply *mtproto.DeliveryRsp, err error) {
//	glog.Infof("DeliveryPushUpdates - request: {%v}", request)
//
//	var seq, replySeq int32
//	now := int32(time.Now().Unix())
//
//	// TODO(@benqi): Check deliver valid!
//	pushDatas := request.GetPushDatas()
//	for _, pushData := range pushDatas {
//		updates := pushData.GetPushData()
//		// pushRawData := updates.Encode()
//		statusList, _ := model.GetOnlineStatusModel().GetOnlineByUserId(pushData.GetPushUserId())
//		switch pushData.GetPushType() {
//		case mtproto.SyncType_SYNC_TYPE_USER:
//			for _, status := range statusList {
//				seq = int32(model.GetSequenceModel().NextSeqId(base.Int64ToString(status.AuthKeyId)))
//				if status.AuthKeyId == request.GetSenderAuthKeyId() {
//					replySeq = seq
//				}
//				updates.SetDate(now)
//				updates.SetSeq(seq)
//
//				//update := &zproto.PushUpdatesNotify{
//				//	AuthKeyId:       status.AuthKeyId,
//				//	SessionId:       status.SessionId,
//				//	NetlibSessionId: status.NetlibSessionId,
//				//	// RawData:         updates.Encode(),
//				//}
//				//
//				//go s.withReadLock(func() {
//				//	s.updates[status.ServerId] <- update
//				//})
//			}
//		case mtproto.SyncType_SYNC_TYPE_USER_NOTME:
//			for _, status := range statusList {
//				seq = int32(model.GetSequenceModel().NextSeqId(base.Int64ToString(status.AuthKeyId)))
//				if status.AuthKeyId == request.GetSenderAuthKeyId() {
//					replySeq = seq
//					continue
//				}
//				updates.SetDate(now)
//				updates.SetSeq(seq)
//
//				//update := &zproto.PushUpdatesNotify{
//				//	AuthKeyId:       status.AuthKeyId,
//				//	SessionId:       status.SessionId,
//				//	NetlibSessionId: status.NetlibSessionId,
//				//	// RawData:         updates.Encode(),
//				//}
//				//
//				//go s.withReadLock(func() {
//				//	s.updates[status.ServerId] <- update
//				//})
//			}
//		case mtproto.SyncType_SYNC_TYPE_AUTH_KEY:
//			for _, status := range statusList {
//				seq = int32(model.GetSequenceModel().NextSeqId(base.Int64ToString(status.AuthKeyId)))
//				if status.AuthKeyId != request.GetSenderAuthKeyId() {
//					replySeq = seq
//				}
//				updates.SetDate(now)
//				updates.SetSeq(seq)
//
//				//update := &zproto.PushUpdatesNotify{
//				//	AuthKeyId:       status.AuthKeyId,
//				//	SessionId:       status.SessionId,
//				//	NetlibSessionId: status.NetlibSessionId,
//				//	// RawData:         updates.Encode(),
//				//}
//				//
//				//go s.withReadLock(func() {
//				//	s.updates[status.ServerId] <- update
//				//})
//			}
//		case mtproto.SyncType_SYNC_TYPE_AUTH_KEY_USERNOTME:
//			for _, status := range statusList {
//				seq = int32(model.GetSequenceModel().NextSeqId(base.Int64ToString(status.AuthKeyId)))
//				if status.AuthKeyId == request.GetSenderAuthKeyId() {
//					replySeq = seq
//					continue
//				}
//				updates.SetDate(now)
//				updates.SetSeq(seq)
//
//				//update := &zproto.PushUpdatesNotify{
//				//	AuthKeyId:       status.AuthKeyId,
//				//	SessionId:       status.SessionId,
//				//	NetlibSessionId: status.NetlibSessionId,
//				//	// RawData:         updates.Encode(),
//				//}
//				//
//				//go s.withReadLock(func() {
//				//	s.updates[status.ServerId] <- update
//				//})
//			}
//		case mtproto.SyncType_SYNC_TYPE_AUTH_KEY_USER:
//			for _, status := range statusList {
//				seq = int32(model.GetSequenceModel().NextSeqId(base.Int64ToString(status.AuthKeyId)))
//				if status.AuthKeyId == request.GetSenderAuthKeyId() {
//					replySeq = seq
//				}
//				updates.SetDate(now)
//				updates.SetSeq(seq)
//
//				//update := &zproto.PushUpdatesNotify{
//				//	AuthKeyId:       status.AuthKeyId,
//				//	SessionId:       status.SessionId,
//				//	NetlibSessionId: status.NetlibSessionId,
//				//	// RawData:         updates.Encode(),
//				//}
//				//
//				//go s.withReadLock(func() {
//				//	s.updates[status.ServerId] <- update
//				//})
//			}
//		default:
//		}
//
//		//ss := make(map[int32][]*model.SessionStatus)
//		//for _, status := range statusList {
//		//	if _, ok := ss[status.ServerId]; !ok {
//		//		ss[status.ServerId] = []*model.SessionStatus{}
//		//	}
//		//	// 会不会出问题？？
//		//	ss[status.ServerId] = append(ss[status.ServerId], status)
//		//}
//		//
//		//for k, ss3 := range ss {
//		//	// glog.Infof("DeliveryUpdates: k: {%v}, v: {%v}", k, ss3)
//		//	go s.withReadLock(func() {
//		//		for _, ss4 := range ss3 {
//		//			update := &zproto.PushUpdatesData{
//		//				AuthKeyId:       ss4.AuthKeyId,
//		//				SessionId:       ss4.SessionId,
//		//				NetlibSessionId: ss4.NetlibSessionId,
//		//				// RawData:         pushRawData,
//		//			}
//		//
//		//
//		//
//		//			s.updates[k] <- update
//		//
//		//		}
//		//	})
//		//}
//
//		// 			updates := pushData.GetPushData()
//		// _ = updates
//	}
//
//	//seq = int32(model.GetSequenceModel().NextSeqId(base.Int64ToString(request.GetSenderAuthKeyId())))
//	reply = &mtproto.DeliveryRsp{
//		Seq:  replySeq,
//		Date: now,
//	}
//
//	glog.Infof("DeliveryPushUpdateShortMessage - reply: %s", logger.JsonDebugData(reply))
//	return
//}
*/
