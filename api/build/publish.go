// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-vela/server/database"
	"github.com/go-vela/server/queue"
	"github.com/go-vela/types"
	"github.com/go-vela/types/library"
	"github.com/sirupsen/logrus"
)

// PublishToQueue is a helper function that publishes a queue item (build, repo, user) to the queue.
func PublishToQueue(ctx context.Context, queue queue.Service, db database.Interface, b *library.Build, r *library.Repo, u *library.User, route string) {
	// convert build, repo, and user into queue item
	item := types.ToItem(b, r, u)

	logrus.Infof("Converting queue item to json for build %d for %s", b.GetNumber(), r.GetFullName())

	byteItem, err := json.Marshal(item)
	if err != nil {
		logrus.Errorf("Failed to convert item to json for build %d for %s: %v", b.GetNumber(), r.GetFullName(), err)

		// error out the build
		CleanBuild(ctx, db, b, nil, nil, err)

		return
	}

	logrus.Infof("Establishing route for build %d for %s", b.GetNumber(), r.GetFullName())

	logrus.Infof("Publishing item for build %d for %s to queue %s", b.GetNumber(), r.GetFullName(), route)

	// push item on to the queue
	err = queue.Push(context.Background(), route, byteItem)
	if err != nil {
		logrus.Errorf("Retrying; Failed to publish build %d for %s: %v", b.GetNumber(), r.GetFullName(), err)

		err = queue.Push(context.Background(), route, byteItem)
		if err != nil {
			logrus.Errorf("Failed to publish build %d for %s: %v", b.GetNumber(), r.GetFullName(), err)

			// error out the build
			CleanBuild(ctx, db, b, nil, nil, err)

			return
		}
	}

	// update fields in build object
	b.SetEnqueued(time.Now().UTC().Unix())

	// update the build in the db to reflect the time it was enqueued
	_, err = db.UpdateBuild(ctx, b)
	if err != nil {
		logrus.Errorf("Failed to update build %d during publish to queue for %s: %v", b.GetNumber(), r.GetFullName(), err)
	}
}
