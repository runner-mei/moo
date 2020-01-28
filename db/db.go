package db

import (
	"database/sql"

	gobatis "github.com/runner-mei/GoBatis"
	"github.com/runner-mei/errors"
	"github.com/runner-mei/log"
	"github.com/runner-mei/moo"
	"go.uber.org/fx"
)

type ArgModelDb struct {
	fx.In
	DrvName string  `name:"drv_models"`
	DB      *sql.DB `name:"models"`
}

type ArgDataDb struct {
	fx.In
	DrvName string  `name:"drv_data"`
	DB      *sql.DB `name:"data"`
}

type ArgModelFactory struct {
	fx.In
	Factory *gobatis.SessionFactory `name:"modelFactory"`
}

type ArgDataFactory struct {
	fx.In
	Factory *gobatis.SessionFactory `name:"dataFactory"`
}

type ArgConstants struct {
	fx.In
	Constants map[string]interface{} `name:"db_constants"`
}

type DbResult struct {
	fx.Out
	Constants            map[string]interface{}  `name:"db_constants"`
	DrvModels            string                  `name:"drv_models"`
	DrvData              string                  `name:"drv_data"`
	Models               *sql.DB                 `name:"models"`
	Data                 *sql.DB                 `name:"data"`
	ModelsSessionFactory *gobatis.SessionFactory `name:"modelFactory"`
	DataSessionFactory   *gobatis.SessionFactory `name:"dataFactory"`
}

func init() {
	moo.On(func() moo.Option {
		return fx.Provide(func(env *moo.Environment, logger log.Logger) (*DbResult, error) {
			drvModels, urlModels := env.Db.Models.Url()
			dbModels, err := sql.Open(drvModels, urlModels)
			if nil != err {
				return nil, errors.Wrap(err, "connect to models database failed")
			}

			drvData, urlData := env.Db.Data.Url()
			dbData, err := sql.Open(drvData, urlData)
			if nil != err {
				dbModels.Close()
				return nil, errors.Wrap(err, "connect to models database failed")
			}

			constants := map[string]interface{}{
				// "discriminator_request": itsm_models.DiscriminatorRequest,
				// "discriminator_task":    itsm_models.DiscriminatorTask,
			}

			tracer := log.NewSQLTracer(logger.Named("gobatis.models"))
			modelFactory, err := gobatis.New(&gobatis.Config{
				Tracer:     tracer,
				TagPrefix:  "xorm",
				TagMapper:  gobatis.TagSplitForXORM,
				Constants:  constants,
				DriverName: drvModels,
				DB:         dbModels,
				XMLPaths: []string{
					"gobatis",
				},
			})
			if err != nil {
				return nil, errors.Wrap(err, "connect to models factory failed")
			}

			tracer = log.NewSQLTracer(logger.Named("gobatis.data"))
			dataFactory, err := gobatis.New(&gobatis.Config{Tracer: tracer,
				TagPrefix:  "xorm",
				TagMapper:  gobatis.TagSplitForXORM,
				Constants:  constants,
				DriverName: drvData,
				DB:         dbData,
				XMLPaths: []string{
					"gobatis",
				},
			})
			if err != nil {
				return nil, errors.Wrap(err, "connect to data factory failed")
			}
			return &DbResult{
				Constants:            constants,
				DrvModels:            drvModels,
				DrvData:              drvData,
				Models:               dbModels,
				Data:                 dbData,
				ModelsSessionFactory: modelFactory,
				DataSessionFactory:   dataFactory,
			}, nil
		})
	})
}
