package informer

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInformerOptionsFromViper(t *testing.T) {
	// Сохраняем исходный Viper и восстанавливаем его после теста
	originalViper := viper.GetViper()
	defer func() {
		viper.Reset()
		*viper.GetViper() = *originalViper
	}()
	
	// Test with default values
	t.Run("default values", func(t *testing.T) {
		viper.Reset()
		
		opts, err := GetInformerOptionsFromViper()
		require.NoError(t, err)
		
		// Проверяем значения по умолчанию для плоской структуры
		defaults := DefaultInformerOptions()
		assert.Equal(t, defaults.Namespace, opts.Namespace)
		assert.Equal(t, defaults.ResyncPeriod, opts.ResyncPeriod)
		assert.Equal(t, defaults.LabelSelector, opts.LabelSelector)
		assert.Equal(t, defaults.FieldSelector, opts.FieldSelector)
		assert.Equal(t, defaults.QPS, opts.QPS)
		assert.Equal(t, defaults.Burst, opts.Burst)
		assert.Equal(t, defaults.Timeout, opts.Timeout)
		assert.Equal(t, defaults.EnableEventLogging, opts.EnableEventLogging)
		assert.Equal(t, defaults.LogLevel, opts.LogLevel)
		assert.Equal(t, defaults.DisableInformer, opts.DisableInformer)
	})
	
	// Test with custom values
	t.Run("custom values", func(t *testing.T) {
		viper.Reset()
		
		// Устанавливаем пользовательские значения в Viper
		viper.Set("informer.namespace", "test-namespace")
		viper.Set("informer.resync_period", 1*time.Minute)
		viper.Set("informer.label_selector", "app=test")
		viper.Set("informer.field_selector", "metadata.name=test")
		viper.Set("kubernetes.qps", 10.0)
		viper.Set("kubernetes.burst", 20)
		viper.Set("kubernetes.timeout", 20*time.Second)
		viper.Set("informer.enable_event_logging", false)
		viper.Set("informer.log_level", "debug")
		viper.Set("kubernetes.disable_informer", true)
		
		opts, err := GetInformerOptionsFromViper()
		require.NoError(t, err)
		
		// Проверяем пользовательские значения
		assert.Equal(t, "test-namespace", opts.Namespace)
		assert.Equal(t, 1*time.Minute, opts.ResyncPeriod)
		assert.Equal(t, "app=test", opts.LabelSelector)
		assert.Equal(t, "metadata.name=test", opts.FieldSelector)
		assert.Equal(t, float32(10.0), opts.QPS)
		assert.Equal(t, 20, opts.Burst)
		assert.Equal(t, 20*time.Second, opts.Timeout)
		assert.False(t, opts.EnableEventLogging)
		assert.Equal(t, "debug", opts.LogLevel)
		assert.True(t, opts.DisableInformer)
	})
}

func TestSetupInformerDefaults(t *testing.T) {
	// Создаем новый экземпляр Viper для теста
	v := viper.New()
	
	// Устанавливаем значения по умолчанию
	SetupInformerDefaults(v)
	
	// Получаем значения по умолчанию для сравнения
	defaults := DefaultInformerOptions()
	
	// Проверяем что значения установлены корректно
	assert.Equal(t, defaults.Namespace, v.GetString("informer.namespace"))
	assert.Equal(t, defaults.ResyncPeriod, v.GetDuration("informer.resync_period"))
	assert.Equal(t, defaults.LabelSelector, v.GetString("informer.label_selector"))
	assert.Equal(t, defaults.FieldSelector, v.GetString("informer.field_selector"))
	assert.Equal(t, float64(defaults.QPS), v.GetFloat64("kubernetes.qps"))
	assert.Equal(t, defaults.Burst, v.GetInt("kubernetes.burst"))
	assert.Equal(t, defaults.Timeout, v.GetDuration("kubernetes.timeout"))
	assert.Equal(t, defaults.EnableEventLogging, v.GetBool("informer.enable_event_logging"))
	assert.Equal(t, defaults.LogLevel, v.GetString("informer.log_level"))
	assert.Equal(t, defaults.DisableInformer, v.GetBool("kubernetes.disable_informer"))
}
