package audited

import (
	"fmt"

	"github.com/ecletus/roles"

	"github.com/moisespsena-go/path-helpers"

	"github.com/ecletus/admin"
	"github.com/ecletus/core/resource"
	"github.com/moisespsena-go/aorm"
)

var (
	pShowAll = roles.PermissionMode(path_helpers.GetCalledDir() + ":show_all")
)

func ModeShowAll() roles.PermissionMode {
	return pShowAll
}

type Audited struct {
	User            *admin.Resource
	FilterByUpdater bool
	Join            aorm.JoinType
	Permission      *roles.Permission
	AdminRole       string
}

func (a *Audited) Setup(res *admin.Resource) *admin.Resource {
	_ = res.Value.(aorm.ModelWithVirtualFields)
	scope := res.FakeScope
	userMS := a.User.FakeScope.ModelStruct()
	fCreator := scope.SetVirtualField("Creator", a.User.Value)
	fCreator.LocalFieldName = "CreatedByID"
	res.Meta(&admin.Meta{
		Name:     "Creator",
		Resource: a.User,
		Config: &admin.SelectOneConfig{
			RemoteDataResource: admin.NewDataResource(a.User),
		},
	})

	if a.AdminRole != "" {
		if a.Permission == nil {
			if res.Permission == nil {
				a.Permission = roles.NewPermission()
			} else {
				a.Permission = res.Permission
			}
		}
		a.Permission.Allow(pShowAll, a.AdminRole)
	}

	scope.SetVirtualField("Updater", a.User.Value)
	events := []resource.DBActionEvent{resource.E_DB_ACTION_FIND_MANY.Before(), resource.E_DB_ACTION_FIND_ONE.Before()}
	join := a.Join

	if a.FilterByUpdater {
		join |= aorm.JoinInner
		events = append(events, resource.E_DB_ACTION_COUNT.Before())
	}

	_ = res.OnDBAction(func(e *resource.DBEvent) {
		if a.Permission != nil {
			userID := e.DB().GetCurrentUserID()
			if !a.Permission.HasPermissionS(pShowAll, e.Context.Roles...) {
				e.SetDB(e.DB().Where(aorm.IQ("{}.created_by_id = ?"), userID))
			}
		}
		options := &aorm.InlinePreloadOptions{
			Join: join,
		}
		options.Where(func(info *aorm.InlinePreloadInfo, replace func(query interface{}, args ...interface{})) {
			query := fmt.Sprintf("{}.%s = %s.created_by_id", userMS.PrimaryFields[0].DBName, info.ParentScope.TableName())
			replace(aorm.IQ(query))
			if a.FilterByUpdater {
				userID := scope.GetCurrentUserID()
				if a.FilterByUpdater {
					info.Conditions.Where("updated_by_id = ?", userID)
				}
			}
		})

		e.SetDB(e.DB().InlinePreload("Creator", options))
	}, events...)
	res.Meta(&admin.Meta{Name: "CreatedByID", Enabled: func(recorde interface{}, context *admin.Context, meta *admin.Meta) bool {
		return false
	}})
	res.Meta(&admin.Meta{Name: "UpdatedByID", Enabled: func(recorde interface{}, context *admin.Context, meta *admin.Meta) bool {
		return false
	}})
	return res
}
